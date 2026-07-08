//go:build gascity_doltlite_lib

package beads

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gastownhall/gascity/internal/beadmeta"
)

// DoltliteReadStore serves hot read paths in-process for bd/doltlite stores.
// Wisps-tier writes stay in-process so the controller does not contend with
// external bd subprocesses for DoltLite's file lock.
type DoltliteReadStore struct {
	*BdStore
	db              *sql.DB
	dbPath          string
	scopeRoot       string
	attachments     []doltliteAttachment
	orderRunMu      sync.Mutex
	orderRunLastRun map[string]time.Time
	orderRunOpen    map[string]bool
	orderRunHash    string
	sessionMu       sync.Mutex
	sessionCache    []Bead
	sessionHash     string
	readyMu         sync.Mutex
	readyCache      map[string][]Bead
	readyHash       string
	columnsMu       sync.Mutex
	columnsCache    map[string]map[string]bool
}

func (s *DoltliteReadStore) NeedsSessionTypeFallback() bool { return true }

type doltliteMetadata struct {
	Backend           string               `json:"backend"`
	Database          string               `json:"database"`
	DoltDatabase      string               `json:"dolt_database"`
	AttachedDatabases []doltliteAttachment `json:"attached_databases,omitempty"`
}

type doltliteAttachment struct {
	Alias string `json:"alias"`
	Path  string `json:"path"`
}

type doltliteTableSet struct {
	issues string
	labels string
	deps   string
	// wisps marks the wisp-backed table set. Snapshots written before the
	// storage-flag columns existed have no per-row discriminator there, in
	// which case every row in the set is ephemeral.
	wisps bool
}

var (
	doltliteIssueTables = doltliteTableSet{issues: "issues", labels: "labels", deps: "dependencies"}
	doltliteWispTables  = doltliteTableSet{issues: "wisps", labels: "wisp_labels", deps: "wisp_dependencies", wisps: true}
)

// doltliteTableSetsForMode maps a TierMode to the storage tables that can hold
// matching rows. TierIssues spans both tables because non-ephemeral
// (no_history) wisps rows belong to the durable tier (query.go TierMode
// contract, #3444); queryIssueTable applies the per-row storage-flag filter.
func doltliteTableSetsForMode(mode TierMode) []doltliteTableSet {
	switch mode {
	case TierWisps:
		return []doltliteTableSet{doltliteWispTables}
	default: // TierIssues, TierBoth
		return []doltliteTableSet{doltliteIssueTables, doltliteWispTables}
	}
}

func (s *DoltliteReadStore) doltliteReadyIssueWhere(tables doltliteTableSet) (string, []any, error) {
	issueTarget, err := s.doltliteDependsOnExprForTable(tables.deps, "d")
	if err != nil {
		return "", nil, err
	}
	wispTarget, err := s.doltliteNullableExprForColumns(tables.deps, "d", []string{"depends_on_wisp_id"})
	if err != nil {
		return "", nil, err
	}
	readyWhere, readyArgs := doltliteReadyIssueWhere(tables, s.tableExists(doltliteWispTables.issues), issueTarget, wispTarget)
	return readyWhere, readyArgs, nil
}

func doltliteReadyIssueWhere(tables doltliteTableSet, includeWispTargets bool, issueTargetExpr, wispTargetExpr string) (string, []any) {
	typePredicate, args := doltliteIssueTypeNotInPredicate("i")
	blockingTypes := make([]string, 0, len(readyBlockingDependencyTypes))
	for typ := range readyBlockingDependencyTypes {
		blockingTypes = append(blockingTypes, typ)
	}
	sort.Strings(blockingTypes)
	blockingPlaceholders := strings.TrimRight(strings.Repeat("?,", len(blockingTypes)), ",")
	for _, typ := range blockingTypes {
		args = append(args, typ)
	}

	depType := "COALESCE(NULLIF(d.type, ''), 'blocks')"
	blockerJoins := "LEFT JOIN " + tables.issues + " blocker_issue ON blocker_issue.id = " + issueTargetExpr
	blockerStatus := "COALESCE(blocker_issue.status, '')"
	issueFailedPredicate, issueFailedArgs := doltliteMetadataEqualsPredicate("blocker_issue", beadmeta.OutcomeMetadataKey, "fail")
	blockerFailed := issueFailedPredicate
	failedArgs := issueFailedArgs
	if includeWispTargets {
		blockerJoins += "\n\t\t\tLEFT JOIN " + doltliteWispTables.issues + " blocker_wisp ON blocker_wisp.id = " + wispTargetExpr
		blockerStatus = "CASE WHEN " + wispTargetExpr + " IS NOT NULL THEN COALESCE(blocker_wisp.status, '') ELSE COALESCE(blocker_issue.status, '') END"
		wispFailedPredicate, wispFailedArgs := doltliteMetadataEqualsPredicate("blocker_wisp", beadmeta.OutcomeMetadataKey, "fail")
		blockerFailed = "CASE WHEN " + wispTargetExpr + " IS NOT NULL THEN (" + wispFailedPredicate + ") ELSE (" + issueFailedPredicate + ") END"
		failedArgs = append(wispFailedArgs, issueFailedArgs...)
	}
	args = append(args, failedArgs...)

	return strings.Join([]string{
		typePredicate,
		`NOT EXISTS (
				SELECT 1 FROM ` + tables.deps + ` d
				` + blockerJoins + `
				WHERE d.issue_id = i.id AND ` + depType + ` IN (` + blockingPlaceholders + `) AND (` + blockerStatus + ` != 'closed' OR ` + blockerFailed + `)
			)`,
	}, " AND "), args
}

func doltliteIssueTypeNotInPredicate(alias string) (string, []any) {
	excluded := make([]string, 0, len(readyExcludeTypes))
	for typ := range readyExcludeTypes {
		excluded = append(excluded, typ)
	}
	sort.Strings(excluded)
	placeholders := strings.TrimRight(strings.Repeat("?,", len(excluded)), ",")
	args := make([]any, 0, len(excluded))
	for _, typ := range excluded {
		args = append(args, typ)
	}
	return "COALESCE(" + alias + ".issue_type, '') NOT IN (" + placeholders + ")", args
}

func NewDoltliteReadStore(dir string, backing *BdStore) (*DoltliteReadStore, error) {
	meta, err := readDoltliteMetadata(dir)
	if err != nil {
		return nil, err
	}
	dbName := strings.TrimSpace(meta.DoltDatabase)
	if dbName == "" || dbName == "doltlite" {
		dbName = strings.TrimSpace(meta.Database)
	}
	if dbName == "" || dbName == "doltlite" {
		dbName = "hq"
	}
	dbPath := filepath.Join(dir, ".beads", "doltlite", dbName+".db")
	if _, err := os.Stat(dbPath); err != nil {
		if !isDoltliteMetadata(meta.Backend, meta.Database) {
			return nil, fmt.Errorf("not a doltlite beads store")
		}
		return nil, err
	}
	db, err := sql.Open(doltliteSQLDriverName, "file:"+dbPath+"?mode=ro&_busy_timeout=10000")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	attachments, err := prepareDoltliteAttachments(db, dir, meta.AttachedDatabases, false)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	store := &DoltliteReadStore{BdStore: backing, db: db, dbPath: dbPath, scopeRoot: dir, attachments: attachments}
	store.warmSchemaCache()
	return store, nil
}

func readDoltliteMetadata(dir string) (doltliteMetadata, error) {
	var meta doltliteMetadata
	data, err := os.ReadFile(filepath.Join(dir, ".beads", "metadata.json"))
	if err != nil {
		return meta, err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

func prepareDoltliteAttachments(db *sql.DB, scopeRoot string, attachments []doltliteAttachment, createMissing bool) ([]doltliteAttachment, error) {
	out := make([]doltliteAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		alias := strings.TrimSpace(attachment.Alias)
		path := strings.TrimSpace(attachment.Path)
		if alias == "" || path == "" {
			continue
		}
		if !validDoltliteAttachmentAlias(alias) {
			return nil, fmt.Errorf("doltlite attachment alias %q is invalid", alias)
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(scopeRoot, path)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("creating doltlite attachment dir %s: %w", filepath.Dir(path), err)
		}
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) && !createMissing {
				continue
			}
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("stat doltlite attachment %s: %w", path, err)
			}
		}
		if _, err := db.Exec(`ATTACH DATABASE ? AS `+quoteSQLiteIdentifier(alias), path); err != nil {
			return nil, fmt.Errorf("attaching doltlite sqlite database %s as %s: %w", path, alias, err)
		}
		out = append(out, doltliteAttachment{Alias: alias, Path: path})
	}
	return out, nil
}

func validDoltliteAttachmentAlias(alias string) bool {
	if alias == "" {
		return false
	}
	for i, r := range alias {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func (s *DoltliteReadStore) CloseStore() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *DoltliteReadStore) Get(id string) (Bead, error) {
	beads, err := s.queryIssues(ListQuery{AllowScan: true, IncludeClosed: true, TierMode: TierBoth}, "i.id = ?", []any{id}, 1)
	if err != nil {
		return Bead{}, err
	}
	if len(beads) == 0 {
		return Bead{}, fmt.Errorf("getting bead %q: %w", id, ErrNotFound)
	}
	return beads[0], nil
}

func (s *DoltliteReadStore) GetSessionBead(id string) (Bead, error) {
	sessions, err := s.ListSessionBeads()
	if err == nil {
		for _, session := range sessions {
			if session.ID == id || session.Metadata["session_name"] == id {
				return session, nil
			}
		}
	}
	beads, err := s.queryIssues(ListQuery{
		AllowScan:     true,
		IncludeClosed: true,
		SkipLabels:    true,
		TierMode:      TierBoth,
	}, "i.id = ?", []any{id}, 1)
	if err != nil {
		return Bead{}, err
	}
	if len(beads) == 0 {
		return Bead{}, fmt.Errorf("getting session bead %q: %w", id, ErrNotFound)
	}
	if beads[0].Type != "session" && beads[0].Type != "" {
		return Bead{}, fmt.Errorf("getting session bead %q: %w", id, ErrNotFound)
	}
	if beads[0].Type == "" {
		return s.Get(id)
	}
	return beads[0], nil
}

func (s *DoltliteReadStore) ListSessionBeads() ([]Bead, error) {
	hash, err := s.currentDoltHash()
	if err != nil {
		return nil, err
	}
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if hash != "" && hash == s.sessionHash && s.sessionCache != nil {
		return cloneBeads(s.sessionCache), nil
	}
	rows, err := s.queryIssues(ListQuery{
		Type:       "session",
		SkipLabels: true,
		TierMode:   TierBoth,
	}, "", nil, 0)
	if err != nil {
		return nil, err
	}
	s.sessionCache = cloneBeads(rows)
	s.sessionHash = hash
	return rows, nil
}

func (s *DoltliteReadStore) List(query ListQuery) ([]Bead, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	if !query.HasFilter() && !query.AllowScan {
		return nil, fmt.Errorf("bd list: %w", ErrQueryRequiresScan)
	}
	return s.queryIssues(query, "", nil, query.Limit)
}

func (s *DoltliteReadStore) ListOpen(status ...string) ([]Bead, error) {
	query := ListQuery{AllowScan: true}
	if len(status) > 0 {
		query.Status = strings.TrimSpace(status[0])
	}
	return s.List(query)
}

func (s *DoltliteReadStore) Children(parentID string, opts ...QueryOpt) ([]Bead, error) {
	return s.List(ListQuery{
		ParentID:      parentID,
		IncludeClosed: HasOpt(opts, IncludeClosed),
		AllowScan:     true,
		Sort:          SortCreatedAsc,
		TierMode:      TierModeFromOpts(opts),
	})
}

func (s *DoltliteReadStore) ListByLabel(label string, limit int, opts ...QueryOpt) ([]Bead, error) {
	return s.List(ListQuery{
		Label:         label,
		Limit:         limit,
		IncludeClosed: HasOpt(opts, IncludeClosed),
		Sort:          SortCreatedDesc,
		TierMode:      TierModeFromOpts(opts),
	})
}

func (s *DoltliteReadStore) ListByAssignee(assignee, status string, limit int) ([]Bead, error) {
	return s.List(ListQuery{
		Assignee: assignee,
		Status:   status,
		Limit:    limit,
	})
}

func (s *DoltliteReadStore) ListByMetadata(filters map[string]string, limit int, opts ...QueryOpt) ([]Bead, error) {
	return s.List(ListQuery{
		Metadata:      filters,
		Limit:         limit,
		IncludeClosed: HasOpt(opts, IncludeClosed),
		Sort:          SortCreatedDesc,
		TierMode:      TierModeFromOpts(opts),
	})
}

func (s *DoltliteReadStore) Ready(query ...ReadyQuery) ([]Bead, error) {
	rq := readyQueryFromArgs(query)
	assignees := readyQueryAssignees(rq)
	cacheKey := fmt.Sprintf("%s\x00%d\x00%d", strings.Join(assignees, "\x1f"), rq.Limit, rq.TierMode)
	hash, err := s.currentDoltHash()
	if err != nil {
		return nil, err
	}
	s.readyMu.Lock()
	if hash != "" && hash == s.readyHash && s.readyCache != nil {
		if cached, ok := s.readyCache[cacheKey]; ok {
			s.readyMu.Unlock()
			return cloneBeads(cached), nil
		}
	}
	s.readyMu.Unlock()

	q := ListQuery{Status: "open", AllowScan: true, IncludeClosed: false, Limit: 0, SkipLabels: true}
	switch len(assignees) {
	case 0:
	case 1:
		q.Assignee = assignees[0]
	default:
		q.Assignees = assignees
	}
	if rq.Limit > 0 {
		q.Limit = rq.Limit
	}
	sets := []doltliteTableSet{doltliteIssueTables}
	switch rq.TierMode {
	case TierBoth:
		sets = []doltliteTableSet{doltliteIssueTables, doltliteWispTables}
		q.TierMode = TierBoth
	case TierWisps:
		sets = []doltliteTableSet{doltliteWispTables}
		q.TierMode = TierWisps
	default:
		// Raw Ready stays on the durable issues table even though aligned
		// TierIssues List reads span no-history wisps (#3444): claimable work
		// remains history-backed unless the caller explicitly asks for wisps.
		q.TierMode = TierIssues
	}
	orderBy := "ORDER BY COALESCE(i.priority, 2) ASC, i.created_at ASC, i.id ASC"
	out := make([]Bead, 0)
	seen := make(map[string]struct{})
	for _, tables := range sets {
		readyWhere, readyArgs, err := s.doltliteReadyIssueWhere(tables)
		if err != nil {
			return nil, err
		}
		rows, err := s.queryIssueTable(q, tables, readyWhere, readyArgs, q.Limit, orderBy)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if _, ok := seen[row.ID]; ok {
				continue
			}
			seen[row.ID] = struct{}{}
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if pi, pj := doltliteReadyPriority(out[i]), doltliteReadyPriority(out[j]); pi != pj {
			return pi < pj
		}
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	if rq.Limit > 0 && len(out) > rq.Limit {
		out = out[:rq.Limit]
	}
	s.readyMu.Lock()
	if hash != "" {
		if hash != s.readyHash || s.readyCache == nil {
			s.readyHash = hash
			s.readyCache = make(map[string][]Bead)
		}
		s.readyCache[cacheKey] = cloneBeads(out)
	}
	s.readyMu.Unlock()
	return out, nil
}

func doltliteReadyPriority(b Bead) int {
	if b.Priority == nil {
		return 2
	}
	return *b.Priority
}

func (s *DoltliteReadStore) LastOrderRun(name string) (time.Time, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return time.Time{}, nil
	}
	hash, err := s.currentDoltHash()
	if err != nil {
		return time.Time{}, err
	}
	s.orderRunMu.Lock()
	defer s.orderRunMu.Unlock()
	if s.orderRunLastRun == nil || hash == "" || hash != s.orderRunHash {
		lastRun, openRuns, err := s.loadOrderRuns()
		if err != nil {
			return time.Time{}, err
		}
		s.orderRunLastRun = lastRun
		s.orderRunOpen = openRuns
		s.orderRunHash = hash
	}
	return s.orderRunLastRun[name], nil
}

func (s *DoltliteReadStore) loadOrderRuns() (map[string]time.Time, map[string]bool, error) {
	rows, err := s.db.Query(`SELECT l.label, MAX(i.created_at), MAX(CASE WHEN i.status != 'closed' THEN 1 ELSE 0 END)
		FROM labels l
		JOIN issues i ON i.id = l.issue_id
		WHERE l.label >= 'order-run:' AND l.label < 'order-run;'
		GROUP BY l.label`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	lastRun := make(map[string]time.Time)
	openRuns := make(map[string]bool)
	for rows.Next() {
		var label string
		var createdRaw any
		var open int
		if err := rows.Scan(&label, &createdRaw, &open); err != nil {
			return nil, nil, err
		}
		name := strings.TrimPrefix(label, "order-run:")
		if name != "" {
			lastRun[name] = parseDBTime(createdRaw).Truncate(time.Second)
			openRuns[name] = open > 0
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return lastRun, openRuns, nil
}

func (s *DoltliteReadStore) HasOpenOrderRun(name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	hash, err := s.currentDoltHash()
	if err != nil {
		return false, err
	}
	s.orderRunMu.Lock()
	defer s.orderRunMu.Unlock()
	if s.orderRunOpen == nil || hash == "" || hash != s.orderRunHash {
		lastRun, openRuns, err := s.loadOrderRuns()
		if err != nil {
			return false, err
		}
		s.orderRunLastRun = lastRun
		s.orderRunOpen = openRuns
		s.orderRunHash = hash
	}
	return s.orderRunOpen[name], nil
}

func (s *DoltliteReadStore) currentDoltHash() (string, error) {
	issueCount, issueUpdated, err := s.tableFingerprint("issues", true)
	if err != nil {
		return "", fmt.Errorf("doltlite issues fingerprint: %w", err)
	}
	wispCount, wispUpdated, err := s.tableFingerprint("wisps", false)
	if err != nil {
		return "", fmt.Errorf("doltlite wisps fingerprint: %w", err)
	}
	labelCount, err := s.tableCount("labels", true)
	if err != nil {
		return "", fmt.Errorf("doltlite labels fingerprint: %w", err)
	}
	wispLabelCount, err := s.tableCount("wisp_labels", false)
	if err != nil {
		return "", fmt.Errorf("doltlite wisp labels fingerprint: %w", err)
	}
	depCount, err := s.tableCount("dependencies", true)
	if err != nil {
		return "", fmt.Errorf("doltlite dependencies fingerprint: %w", err)
	}
	wispDepCount, err := s.tableCount("wisp_dependencies", false)
	if err != nil {
		return "", fmt.Errorf("doltlite wisp dependencies fingerprint: %w", err)
	}

	return fmt.Sprintf("issues=%d:%s;wisps=%d:%s;labels=%d:%d;deps=%d:%d",
		issueCount, issueUpdated, wispCount, wispUpdated, labelCount, wispLabelCount, depCount, wispDepCount), nil
}

func (s *DoltliteReadStore) tableFingerprint(table string, required bool) (int64, string, error) {
	if !s.tableExists(table) {
		if required {
			return 0, "", fmt.Errorf("missing table %q", table)
		}
		return 0, "", nil
	}
	var count int64
	var maxUpdated sql.NullString
	if err := s.db.QueryRow("SELECT COUNT(*), MAX(updated_at) FROM "+table).Scan(&count, &maxUpdated); err != nil {
		return 0, "", err
	}
	if !maxUpdated.Valid {
		return count, "", nil
	}
	return count, strings.TrimSpace(maxUpdated.String), nil
}

func (s *DoltliteReadStore) tableCount(table string, required bool) (int64, error) {
	if !s.tableExists(table) {
		if required {
			return 0, fmt.Errorf("missing table %q", table)
		}
		return 0, nil
	}
	var count int64
	if err := s.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *DoltliteReadStore) resetOrderRunCache() {
	s.orderRunMu.Lock()
	defer s.orderRunMu.Unlock()
	s.orderRunLastRun = nil
	s.orderRunOpen = nil
	s.orderRunHash = ""
	s.sessionMu.Lock()
	s.sessionCache = nil
	s.sessionHash = ""
	s.sessionMu.Unlock()
	s.readyMu.Lock()
	s.readyCache = nil
	s.readyHash = ""
	s.readyMu.Unlock()
}

func (s *DoltliteReadStore) Create(b Bead) (Bead, error) {
	ephemeral, noHistory, err := effectiveStorageFlags(b, StorageDefault)
	if err != nil {
		return Bead{}, fmt.Errorf("doltlite create: %w", err)
	}
	if ephemeral && noHistory {
		return Bead{}, fmt.Errorf("doltlite create: ephemeral and no-history storage are mutually exclusive")
	}
	if ephemeral || noHistory {
		created, err := s.createWisp(b, ephemeral, noHistory)
		if err == nil {
			s.resetOrderRunCache()
		}
		return created, err
	}
	created, err := s.BdStore.Create(b)
	if err == nil && hasOrderRunLabel(created.Labels) {
		s.resetOrderRunCache()
	}
	return created, err
}

func (s *DoltliteReadStore) createWisp(b Bead, ephemeral, noHistory bool) (Bead, error) {
	if strings.TrimSpace(b.Title) == "" {
		return Bead{}, fmt.Errorf("doltlite create: title is required")
	}
	if b.ParentID != "" || len(b.Needs) > 0 || len(b.Dependencies) > 0 {
		return Bead{}, fmt.Errorf("doltlite create: direct wisp create does not support dependencies")
	}
	created := b
	created.ID = strings.TrimSpace(created.ID)
	if created.ID == "" {
		created.ID = s.nextWispID()
	}
	created.Status = strings.TrimSpace(created.Status)
	if created.Status == "" {
		created.Status = "open"
	}
	created.Type = strings.TrimSpace(created.Type)
	if created.Type == "" {
		created.Type = "task"
	}
	if created.Priority == nil {
		priority := 2
		created.Priority = &priority
	}
	now := time.Now().UTC()
	if created.CreatedAt.IsZero() {
		created.CreatedAt = now
	}
	if created.UpdatedAt.IsZero() {
		created.UpdatedAt = created.CreatedAt
	}
	created.Ephemeral = ephemeral
	created.NoHistory = noHistory
	metadata := maps.Clone(created.Metadata)
	if created.From != "" {
		if metadata == nil {
			metadata = make(map[string]string, 1)
		}
		if metadata["from"] == "" {
			metadata["from"] = created.From
		}
	}
	if metadata == nil {
		metadata = map[string]string{}
	}
	rawMetadata, err := json.Marshal(metadata)
	if err != nil {
		return Bead{}, fmt.Errorf("doltlite create: marshaling metadata: %w", err)
	}
	created.Metadata = metadata

	err = s.runWispWrite(func() error {
		db, err := s.openWritableDB()
		if err != nil {
			return err
		}
		defer db.Close() //nolint:errcheck // best-effort cleanup
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()

		columns := []string{"id", "title", "status", "issue_type", "priority", "created_at", "updated_at", "assignee", "description", "design", "acceptance_criteria", "notes", "metadata"}
		values := []any{created.ID, created.Title, created.Status, created.Type, *created.Priority, doltliteSQLiteTime(created.CreatedAt), doltliteSQLiteTime(created.UpdatedAt), created.Assignee, created.Description, "", "", "", string(rawMetadata)}
		if s.columnExists("wisps", "ephemeral") {
			columns = append(columns, "ephemeral")
			values = append(values, boolToSQLInt(ephemeral))
		}
		if s.columnExists("wisps", "no_history") {
			columns = append(columns, "no_history")
			values = append(values, boolToSQLInt(noHistory))
		}
		placeholders := strings.TrimRight(strings.Repeat("?,", len(columns)), ",")
		if _, err := tx.Exec(`INSERT INTO wisps (`+strings.Join(columns, ",")+`) VALUES (`+placeholders+`)`, values...); err != nil {
			return fmt.Errorf("inserting wisp %q: %w", created.ID, err)
		}
		if s.tableExists("wisp_labels") {
			for _, label := range created.Labels {
				label = strings.TrimSpace(label)
				if label == "" {
					continue
				}
				if _, err := tx.Exec(`INSERT INTO wisp_labels (issue_id, label) VALUES (?, ?)`, created.ID, label); err != nil {
					return fmt.Errorf("inserting wisp label %q for %q: %w", label, created.ID, err)
				}
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		committed = true
		return nil
	})
	if err != nil {
		return Bead{}, fmt.Errorf("doltlite create: %w", err)
	}
	return created, nil
}

func (s *DoltliteReadStore) nextWispID() string {
	prefix := ""
	if s.BdStore != nil {
		prefix = s.BdStore.IDPrefix()
	}
	if prefix == "" {
		prefix = "gc"
	}
	var buf [5]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s-wisp-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-wisp-" + hex.EncodeToString(buf[:])
}

func boolToSQLInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func isWispTierBead(b Bead) bool {
	return b.Ephemeral || b.NoHistory
}

func hasOrderRunLabel(labels []string) bool {
	for _, label := range labels {
		if strings.HasPrefix(label, "order-run:") {
			return true
		}
	}
	return false
}

func (s *DoltliteReadStore) Update(id string, opts UpdateOpts) error {
	current, err := s.doltliteWriteBead(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("updating bead %q: %w", id, ErrNotFound)
		}
		return fmt.Errorf("updating bead %q: %w", id, err)
	}
	if isWispTierBead(current) {
		err := s.updateWisp(id, opts)
		if err == nil {
			s.resetOrderRunCache()
		}
		return err
	}
	err = s.updateIssue(id, opts)
	if err == nil {
		s.resetOrderRunCache()
	}
	return err
}

func (s *DoltliteReadStore) Close(id string) error {
	current, err := s.doltliteWriteBead(id)
	if err != nil {
		return err
	}
	if isWispTierBead(current) {
		err = s.updateWispStatus(id, "closed")
		if err == nil {
			s.resetOrderRunCache()
		}
		return err
	}
	err = s.updateIssueStatus(id, "closed")
	if err == nil {
		s.resetOrderRunCache()
	}
	return err
}

func (s *DoltliteReadStore) CloseAll(ids []string, metadata map[string]string) (int, error) {
	closed := 0
	for _, id := range ids {
		if len(metadata) > 0 {
			if err := s.SetMetadataBatch(id, metadata); err != nil {
				return closed, err
			}
		}
		current, err := s.doltliteWriteBead(id)
		if err != nil {
			return closed, err
		}
		if current.Status == "closed" {
			continue
		}
		if isWispTierBead(current) {
			if err := s.updateWispStatus(id, "closed"); err != nil {
				return closed, err
			}
		} else if err := s.updateIssueStatus(id, "closed"); err != nil {
			return closed, err
		}
		closed++
	}
	if closed > 0 {
		s.resetOrderRunCache()
	}
	return closed, nil
}

func (s *DoltliteReadStore) Reopen(id string) error {
	current, err := s.doltliteWriteBead(id)
	if err != nil {
		return err
	}
	if isWispTierBead(current) {
		err = s.updateWispStatus(id, "open")
		if err == nil {
			s.resetOrderRunCache()
		}
		return err
	}
	err = s.updateIssueStatus(id, "open")
	if err == nil {
		s.resetOrderRunCache()
	}
	return err
}

func (s *DoltliteReadStore) Delete(id string) error {
	current, err := s.doltliteWriteBead(id)
	if err != nil {
		return fmt.Errorf("deleting bead %q: %w", id, err)
	}
	if isWispTierBead(current) {
		err := s.deleteWisp(id)
		if err == nil {
			s.resetOrderRunCache()
		}
		return err
	}
	err = s.deleteIssue(id)
	if err == nil {
		s.resetOrderRunCache()
	}
	return err
}

func (s *DoltliteReadStore) SetMetadataBatch(id string, kvs map[string]string) error {
	if len(kvs) == 0 {
		return nil
	}
	var current Bead
	var changed map[string]string
	err := s.runWispWrite(func() error {
		var err error
		current, err = s.doltliteWriteBead(id)
		if err != nil {
			return err
		}
		changed = changedMetadata(current.Metadata, kvs)
		if len(changed) == 0 {
			return nil
		}
		if isWispTierBead(current) {
			return s.updateWispMetadataLocked(id, changed)
		}
		return s.updateIssueMetadataLocked(id, changed)
	})
	if err != nil {
		return fmt.Errorf("setting metadata on %q: %w", id, err)
	}
	if len(changed) == 0 {
		return nil
	}
	s.resetOrderRunCache()
	return nil
}

func (s *DoltliteReadStore) SetMetadata(id, key, value string) error {
	return s.SetMetadataBatch(id, map[string]string{key: value})
}

func changedMetadata(current, updates map[string]string) map[string]string {
	changed := make(map[string]string, len(updates))
	for k, v := range updates {
		if current[k] != v {
			changed[k] = v
		}
	}
	return changed
}

func (s *DoltliteReadStore) updateWisp(id string, opts UpdateOpts) error {
	return s.updateDoltliteBead(doltliteWispTables, id, opts)
}

func (s *DoltliteReadStore) updateIssue(id string, opts UpdateOpts) error {
	return s.updateDoltliteBead(doltliteIssueTables, id, opts)
}

func (s *DoltliteReadStore) updateDoltliteBead(tables doltliteTableSet, id string, opts UpdateOpts) error {
	if opts.Title == nil &&
		opts.Status == nil &&
		opts.Type == nil &&
		opts.Priority == nil &&
		opts.Description == nil &&
		opts.ParentID == nil &&
		opts.Assignee == nil &&
		len(opts.Labels) == 0 &&
		len(opts.RemoveLabels) == 0 &&
		len(opts.Metadata) == 0 {
		return nil
	}
	return s.runWispWrite(func() error {
		return s.updateDoltliteBeadLocked(tables, id, opts)
	})
}

func (s *DoltliteReadStore) updateWispLocked(id string, opts UpdateOpts) error {
	return s.updateDoltliteBeadLocked(doltliteWispTables, id, opts)
}

func (s *DoltliteReadStore) updateDoltliteBeadLocked(tables doltliteTableSet, id string, opts UpdateOpts) error {
	db, err := s.openWritableDB()
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck // best-effort cleanup
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	updates := make([]string, 0, 8)
	args := make([]any, 0, 10)
	if opts.Title != nil {
		updates = append(updates, "title = ?")
		args = append(args, *opts.Title)
	}
	if opts.Status != nil {
		updates = append(updates, "status = ?")
		args = append(args, *opts.Status)
	}
	if opts.Type != nil {
		updates = append(updates, "issue_type = ?")
		args = append(args, *opts.Type)
	}
	if opts.Priority != nil {
		updates = append(updates, "priority = ?")
		args = append(args, *opts.Priority)
	}
	if opts.Description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *opts.Description)
	}
	if opts.Assignee != nil {
		updates = append(updates, "assignee = ?")
		args = append(args, *opts.Assignee)
	}
	if len(opts.Metadata) > 0 {
		raw, err := s.mergeDoltliteMetadata(tx, tables.issues, id, opts.Metadata)
		if err != nil {
			return fmt.Errorf("updating %s metadata %q: %w", tables.issues, id, err)
		}
		updates = append(updates, "metadata = ?")
		args = append(args, string(raw))
	}
	if len(updates) > 0 {
		updates = append(updates, "updated_at = ?")
		args = append(args, time.Now().UTC().Format(time.RFC3339Nano), id)
		res, err := tx.Exec(`UPDATE `+tables.issues+` SET `+strings.Join(updates, ", ")+` WHERE id = ?`, args...)
		if err != nil {
			return fmt.Errorf("updating %s %q: %w", tables.issues, id, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("checking updated %s %q: %w", tables.issues, id, err)
		}
		if n == 0 {
			return fmt.Errorf("updating %s %q: %w", tables.issues, id, ErrNotFound)
		}
	}
	if opts.ParentID != nil {
		if err := s.replaceDoltliteParent(tx, tables, id, *opts.ParentID); err != nil {
			return fmt.Errorf("updating %s parent %q: %w", tables.issues, id, err)
		}
	}
	if len(opts.Labels) > 0 || len(opts.RemoveLabels) > 0 {
		if err := s.updateDoltliteLabels(tx, tables.labels, id, opts.Labels, opts.RemoveLabels); err != nil {
			return fmt.Errorf("updating %s labels %q: %w", tables.issues, id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *DoltliteReadStore) replaceWispParent(tx *sql.Tx, id, parentID string) error {
	return s.replaceDoltliteParent(tx, doltliteWispTables, id, parentID)
}

func (s *DoltliteReadStore) replaceDoltliteParent(tx *sql.Tx, tables doltliteTableSet, id, parentID string) error {
	if _, err := tx.Exec(`DELETE FROM `+tables.deps+` WHERE issue_id = ? AND type = 'parent-child'`, id); err != nil {
		return err
	}
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return nil
	}
	parent, err := s.doltliteWriteBead(parentID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	columns := []string{"issue_id", "type"}
	values := []any{id, "parent-child"}
	targetColumns, targetValues, err := s.doltliteDependencyTargetInsertValues(tables.deps, parentID, parent)
	if err != nil {
		return err
	}
	columns = append(columns, targetColumns...)
	values = append(values, targetValues...)
	placeholders := strings.TrimRight(strings.Repeat("?,", len(columns)), ",")
	_, err = tx.Exec(`INSERT INTO `+tables.deps+` (`+strings.Join(columns, ", ")+`) VALUES (`+placeholders+`)`, values...)
	return err
}

func (s *DoltliteReadStore) deleteIssue(id string) error {
	return s.deleteDoltliteBead(doltliteIssueTables, id)
}

func (s *DoltliteReadStore) deleteWisp(id string) error {
	return s.deleteDoltliteBead(doltliteWispTables, id)
}

func (s *DoltliteReadStore) deleteDoltliteBead(tables doltliteTableSet, id string) error {
	return s.runWispWrite(func() error {
		db, err := s.openWritableDB()
		if err != nil {
			return err
		}
		defer db.Close() //nolint:errcheck // best-effort cleanup
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()
		if s.tableExists(tables.labels) {
			if _, err := tx.Exec(`DELETE FROM `+tables.labels+` WHERE issue_id = ?`, id); err != nil {
				return err
			}
		}
		if s.tableExists(tables.deps) {
			where := `issue_id = ?`
			args := []any{id}
			targetWhere, hasTarget, err := s.doltliteDependsOnEqualPredicate(tables.deps)
			if err != nil {
				return err
			}
			if hasTarget {
				where += ` OR ` + targetWhere
				args = append(args, id)
			}
			if _, err := tx.Exec(`DELETE FROM `+tables.deps+` WHERE `+where, args...); err != nil {
				return err
			}
		}
		otherDeps := doltliteIssueTables.deps
		if tables.deps == doltliteIssueTables.deps {
			otherDeps = doltliteWispTables.deps
		}
		if s.tableExists(otherDeps) {
			where, hasTarget, err := s.doltliteDependsOnEqualPredicate(otherDeps)
			if err != nil {
				return err
			}
			args := []any{}
			if !hasTarget {
				where = `0`
			} else {
				args = append(args, id)
			}
			if _, err := tx.Exec(`DELETE FROM `+otherDeps+` WHERE `+where, args...); err != nil {
				return err
			}
		}
		res, err := tx.Exec(`DELETE FROM `+tables.issues+` WHERE id = ?`, id)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrNotFound
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		committed = true
		return nil
	})
}

func (s *DoltliteReadStore) addWispDependency(id, dep, depType string, target Bead) error {
	if depType = strings.TrimSpace(depType); depType == "" {
		depType = "blocks"
	}
	return s.runWispWrite(func() error {
		db, err := s.openWritableDB()
		if err != nil {
			return err
		}
		defer db.Close() //nolint:errcheck // best-effort cleanup
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()
		targetWhere, hasTarget, err := s.doltliteDependsOnEqualPredicate("wisp_dependencies")
		if err != nil {
			return err
		}
		if !hasTarget {
			return fmt.Errorf("wisp_dependencies has no dependency target columns")
		}
		if depType != "parent-child" {
			if _, err := tx.Exec(`DELETE FROM wisp_dependencies WHERE issue_id = ? AND `+targetWhere+` AND COALESCE(NULLIF(type, ''), 'blocks') != 'parent-child'`, id, dep); err != nil {
				return err
			}
		} else {
			var exists int
			if err := tx.QueryRow(`SELECT COUNT(1) FROM wisp_dependencies WHERE issue_id = ? AND `+targetWhere+` AND type = 'parent-child'`, id, dep).Scan(&exists); err != nil {
				return err
			}
			if exists > 0 {
				if err := tx.Commit(); err != nil {
					return err
				}
				committed = true
				return nil
			}
		}
		columns := []string{"issue_id", "type"}
		values := []any{id, depType}
		targetColumns, targetValues, err := s.doltliteDependencyTargetInsertValues("wisp_dependencies", dep, target)
		if err != nil {
			return err
		}
		columns = append(columns, targetColumns...)
		values = append(values, targetValues...)
		placeholders := strings.TrimRight(strings.Repeat("?,", len(columns)), ",")
		if _, err := tx.Exec(`INSERT INTO wisp_dependencies (`+strings.Join(columns, ", ")+`) VALUES (`+placeholders+`)`, values...); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		committed = true
		return nil
	})
}

func (s *DoltliteReadStore) removeWispDependency(id, dep string) error {
	return s.runWispWrite(func() error {
		db, err := s.openWritableDB()
		if err != nil {
			return err
		}
		defer db.Close() //nolint:errcheck // best-effort cleanup
		targetWhere, hasTarget, err := s.doltliteDependsOnEqualPredicate("wisp_dependencies")
		if err != nil {
			return err
		}
		if !hasTarget {
			return nil
		}
		_, err = db.Exec(`DELETE FROM wisp_dependencies WHERE issue_id = ? AND `+targetWhere, id, dep)
		return err
	})
}

func (s *DoltliteReadStore) doltliteDependsOnEqualPredicate(table string) (string, bool, error) {
	expr, err := s.doltliteDependsOnExprForTable(table, "")
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(expr) == "''" {
		return "", false, nil
	}
	return expr + " = ?", true, nil
}

func (s *DoltliteReadStore) doltliteDependencyTargetInsertValues(table, targetID string, target Bead) ([]string, []any, error) {
	columns, err := s.tableColumns(table)
	if err != nil {
		return nil, nil, err
	}
	outColumns := make([]string, 0, 4)
	outValues := make([]any, 0, 4)
	add := func(column, value string) {
		if columns[column] {
			outColumns = append(outColumns, column)
			outValues = append(outValues, value)
		}
	}

	if columns["depends_on_id"] {
		add("depends_on_id", targetID)
	}
	isExternal := strings.HasPrefix(strings.ToLower(strings.TrimSpace(targetID)), "external:")
	switch {
	case isExternal:
		add("depends_on_external", targetID)
	case isWispTierBead(target):
		add("depends_on_wisp_id", targetID)
	default:
		add("depends_on_issue_id", targetID)
	}
	if len(outColumns) == 0 {
		return nil, nil, fmt.Errorf("%s has no dependency target columns", table)
	}
	return outColumns, outValues, nil
}

func (s *DoltliteReadStore) updateWispLabels(tx *sql.Tx, id string, addLabels, removeLabels []string) error {
	return s.updateDoltliteLabels(tx, "wisp_labels", id, addLabels, removeLabels)
}

func (s *DoltliteReadStore) updateDoltliteLabels(tx *sql.Tx, table, id string, addLabels, removeLabels []string) error {
	for _, label := range removeLabels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		if _, err := tx.Exec(`DELETE FROM `+table+` WHERE issue_id = ? AND label = ?`, id, label); err != nil {
			return err
		}
	}
	for _, label := range addLabels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		if _, err := tx.Exec(`DELETE FROM `+table+` WHERE issue_id = ? AND label = ?`, id, label); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO `+table+` (issue_id, label) VALUES (?, ?)`, id, label); err != nil {
			return err
		}
	}
	return nil
}

func (s *DoltliteReadStore) doltliteWriteBead(id string) (Bead, error) {
	rows, err := s.queryIssues(ListQuery{
		AllowScan:     true,
		IncludeClosed: true,
		SkipLabels:    true,
		TierMode:      TierBoth,
	}, "i.id = ?", []any{id}, 1)
	if err != nil {
		return Bead{}, err
	}
	if len(rows) == 0 {
		return Bead{}, ErrNotFound
	}
	return rows[0], nil
}

func (s *DoltliteReadStore) updateWispMetadata(id string, metadata map[string]string) error {
	return s.runWispWrite(func() error {
		return s.updateWispMetadataLocked(id, metadata)
	})
}

func (s *DoltliteReadStore) updateWispMetadataLocked(id string, metadata map[string]string) error {
	return s.updateDoltliteMetadataLocked("wisps", id, metadata)
}

func (s *DoltliteReadStore) updateIssueMetadataLocked(id string, metadata map[string]string) error {
	return s.updateDoltliteMetadataLocked("issues", id, metadata)
}

func (s *DoltliteReadStore) updateDoltliteMetadataLocked(table, id string, metadata map[string]string) error {
	db, err := s.openWritableDB()
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck // best-effort cleanup
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	raw, err := s.mergeDoltliteMetadata(tx, table, id, metadata)
	if err != nil {
		return err
	}
	res, err := tx.Exec(`UPDATE `+table+` SET metadata = ?, updated_at = ? WHERE id = ?`, raw, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *DoltliteReadStore) mergeWispMetadata(tx *sql.Tx, id string, updates map[string]string) (string, error) {
	return s.mergeDoltliteMetadata(tx, "wisps", id, updates)
}

func (s *DoltliteReadStore) mergeDoltliteMetadata(tx *sql.Tx, table, id string, updates map[string]string) (string, error) {
	current, err := s.loadDoltliteMetadata(tx, table, id)
	if err != nil {
		return "", err
	}
	for key, value := range updates {
		current[key] = value
	}
	raw, err := json.Marshal(current)
	if err != nil {
		return "", fmt.Errorf("encoding wisp metadata: %w", err)
	}
	return string(raw), nil
}

func (s *DoltliteReadStore) loadWispMetadata(tx *sql.Tx, id string) (map[string]string, error) {
	return s.loadDoltliteMetadata(tx, "wisps", id)
}

func (s *DoltliteReadStore) loadDoltliteMetadata(tx *sql.Tx, table, id string) (map[string]string, error) {
	var raw sql.NullString
	if err := tx.QueryRow(`SELECT metadata FROM `+table+` WHERE id = ?`, id).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	metadata := make(map[string]string)
	if raw.Valid && strings.TrimSpace(raw.String) != "" {
		var decoded StringMap
		if err := json.Unmarshal([]byte(raw.String), &decoded); err != nil {
			return nil, fmt.Errorf("parsing wisp metadata: %w", err)
		}
		metadata = map[string]string(decoded)
		if metadata == nil {
			metadata = make(map[string]string)
		}
	}
	return metadata, nil
}

func (s *DoltliteReadStore) updateWispStatus(id, status string) error {
	return s.updateDoltliteStatus("wisps", id, status)
}

func (s *DoltliteReadStore) updateIssueStatus(id, status string) error {
	return s.updateDoltliteStatus("issues", id, status)
}

func (s *DoltliteReadStore) updateDoltliteStatus(table, id, status string) error {
	return s.runWispWrite(func() error {
		return s.updateDoltliteStatusLocked(table, id, status)
	})
}

func (s *DoltliteReadStore) updateWispStatusLocked(id, status string) error {
	return s.updateDoltliteStatusLocked("wisps", id, status)
}

func (s *DoltliteReadStore) updateDoltliteStatusLocked(table, id, status string) error {
	db, err := s.openWritableDB()
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck // best-effort cleanup
	res, err := db.Exec(`UPDATE `+table+` SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *DoltliteReadStore) runWispWrite(fn func() error) error {
	if s.dbPath == "" {
		return fn()
	}
	lockRoot := filepath.Dir(filepath.Dir(s.dbPath))
	lockPath := filepath.Join(lockRoot, ".bd-write.lock")
	return withDoltliteWriteLock(lockRoot, lockPath, func() error {
		var err error
		for attempt := 1; attempt <= bdTransientWriteAttempts; attempt++ {
			err = fn()
			if err == nil || !isBdTransientWriteError(err) || attempt == bdTransientWriteAttempts {
				return err
			}
			time.Sleep(time.Duration(attempt) * 25 * time.Millisecond)
		}
		return err
	})
}

func (s *DoltliteReadStore) openWritableDB() (*sql.DB, error) {
	if s.dbPath == "" {
		return nil, fmt.Errorf("doltlite writable database path is empty")
	}
	db, err := sql.Open(doltliteSQLDriverName, "file:"+s.dbPath+"?mode=rw&_busy_timeout=10000")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := prepareDoltliteAttachments(db, s.scopeRoot, s.attachments, true); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (s *DoltliteReadStore) DepAdd(id, dep, depType string) error {
	current, err := s.doltliteWriteBead(id)
	if err != nil {
		return fmt.Errorf("adding dep %s→%s: %w", id, dep, err)
	}
	if isWispTierBead(current) {
		target, err := s.doltliteWriteBead(dep)
		if err != nil {
			return fmt.Errorf("adding dep %s→%s: %w", id, dep, err)
		}
		err = s.addWispDependency(id, dep, depType, target)
		if err == nil {
			s.resetOrderRunCache()
		}
		return err
	}
	err = s.BdStore.DepAdd(id, dep, depType)
	if err == nil {
		s.resetOrderRunCache()
	}
	return err
}

func (s *DoltliteReadStore) DepRemove(id, dep string) error {
	current, err := s.doltliteWriteBead(id)
	if err != nil {
		return fmt.Errorf("removing dep %s→%s: %w", id, dep, err)
	}
	if isWispTierBead(current) {
		err := s.removeWispDependency(id, dep)
		if err == nil {
			s.resetOrderRunCache()
		}
		return err
	}
	err = s.BdStore.DepRemove(id, dep)
	if err == nil {
		s.resetOrderRunCache()
	}
	return err
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func cloneBeads(values []Bead) []Bead {
	if len(values) == 0 {
		return nil
	}
	out := make([]Bead, len(values))
	for i := range values {
		out[i] = cloneBead(values[i])
	}
	return out
}

func (s *DoltliteReadStore) DepList(id, direction string) ([]Dep, error) {
	if direction == "up" {
		return s.queryDeps(func(table string) string {
			expr, err := s.doltliteDependsOnExprForTable(table, "")
			if err != nil {
				return "''"
			}
			return expr + " = ?"
		}, id)
	}
	return s.queryDeps(func(string) string { return "issue_id = ?" }, id)
}

func (s *DoltliteReadStore) DepListBatch(ids []string) (map[string][]Dep, error) {
	result := make(map[string][]Dep, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	for start := 0; start < len(ids); start += 500 {
		end := start + 500
		if end > len(ids) {
			end = len(ids)
		}
		placeholders := strings.TrimRight(strings.Repeat("?,", end-start), ",")
		args := make([]any, 0, end-start)
		for _, id := range ids[start:end] {
			args = append(args, id)
		}
		for _, table := range []string{"dependencies", "wisp_dependencies"} {
			if table == "wisp_dependencies" && !s.tableExists(table) {
				continue
			}
			dependsOnExpr, err := s.doltliteDependsOnExprForTable(table, "")
			if err != nil {
				return result, err
			}
			rows, err := s.db.Query(`SELECT issue_id, `+dependsOnExpr+`, type FROM `+table+` WHERE issue_id IN (`+placeholders+`)`, args...)
			if err != nil {
				return result, err
			}
			for rows.Next() {
				dep, err := scanDep(rows)
				if err != nil {
					_ = rows.Close()
					return result, err
				}
				result[dep.IssueID] = append(result[dep.IssueID], dep)
			}
			if err := rows.Err(); err != nil {
				_ = rows.Close()
				return result, err
			}
			if err := rows.Close(); err != nil {
				return result, err
			}
		}
	}
	return result, nil
}

func (s *DoltliteReadStore) dependencySnapshotForCache(ids []string) (map[string][]Dep, bool, error) {
	deps, err := s.DepListBatch(ids)
	if err != nil {
		return deps, false, err
	}
	return deps, true, nil
}

func (s *DoltliteReadStore) enrichReadyProjectionForCache(items []Bead) ([]Bead, error) {
	// Native DoltLite snapshots do not carry bd's denormalized is_blocked
	// projection, so cached ready intentionally keeps the nil fallback.
	return items, nil
}

func (s *DoltliteReadStore) queryDeps(whereForTable func(string) string, value string) ([]Dep, error) {
	var deps []Dep
	for _, table := range []string{"dependencies", "wisp_dependencies"} {
		if table == "wisp_dependencies" && !s.tableExists(table) {
			continue
		}
		dependsOnExpr, err := s.doltliteDependsOnExprForTable(table, "")
		if err != nil {
			return nil, err
		}
		rows, err := s.db.Query(`SELECT issue_id, `+dependsOnExpr+`, type FROM `+table+` WHERE `+whereForTable(table), value)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			dep, err := scanDep(rows)
			if err != nil {
				_ = rows.Close()
				return nil, err
			}
			deps = append(deps, dep)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	return deps, nil
}

func doltliteDependsOnExpr() string {
	return "COALESCE(NULLIF(depends_on_id, ''), NULLIF(depends_on_issue_id, ''), NULLIF(depends_on_wisp_id, ''), NULLIF(depends_on_external, ''), '')"
}

func doltliteQualifiedDependsOnExpr(alias string) string {
	return doltliteQualifiedDependsOnExprFromColumns(alias, map[string]bool{
		"depends_on_id":       true,
		"depends_on_issue_id": true,
		"depends_on_wisp_id":  true,
		"depends_on_external": true,
	})
}

func doltliteQualifiedDependsOnExprFromColumns(alias string, columns map[string]bool) string {
	prefix := ""
	if strings.TrimSpace(alias) != "" {
		prefix = alias + "."
	}
	parts := make([]string, 0, 4)
	for _, column := range []string{"depends_on_id", "depends_on_issue_id", "depends_on_wisp_id", "depends_on_external"} {
		if columns[column] {
			parts = append(parts, "NULLIF("+prefix+column+", '')")
		}
	}
	if len(parts) == 0 {
		return "''"
	}
	return "COALESCE(" + strings.Join(parts, ", ") + ", '')"
}

func (s *DoltliteReadStore) doltliteDependsOnExprForTable(table, alias string) (string, error) {
	columns, err := s.tableColumns(table)
	if err != nil {
		return "", err
	}
	return s.doltliteExprFromColumns(alias, columns), nil
}

func (s *DoltliteReadStore) doltliteNullableExprForColumns(table, alias string, candidates []string) (string, error) {
	columns, err := s.tableColumns(table)
	if err != nil {
		return "", err
	}
	prefix := ""
	if strings.TrimSpace(alias) != "" {
		prefix = alias + "."
	}
	parts := make([]string, 0, len(candidates))
	for _, column := range candidates {
		if columns[column] {
			parts = append(parts, "NULLIF("+prefix+column+", '')")
		}
	}
	if len(parts) == 0 {
		return "NULL", nil
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	return "COALESCE(" + strings.Join(parts, ", ") + ")", nil
}

func (s *DoltliteReadStore) doltliteExprFromColumns(alias string, columns map[string]bool) string {
	return doltliteQualifiedDependsOnExprFromColumns(alias, columns)
}

func (s *DoltliteReadStore) tableColumns(name string) (map[string]bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return map[string]bool{}, nil
	}
	s.columnsMu.Lock()
	if s.columnsCache == nil {
		s.columnsCache = make(map[string]map[string]bool)
	}
	if columns, ok := s.columnsCache[name]; ok {
		out := make(map[string]bool, len(columns))
		for column := range columns {
			out[column] = true
		}
		s.columnsMu.Unlock()
		return out, nil
	}
	s.columnsMu.Unlock()

	rows, err := s.db.Query(`PRAGMA table_info(` + quoteSQLiteIdentifier(name) + `)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck // best-effort cleanup
	columns := make(map[string]bool)
	for rows.Next() {
		var (
			cid        int
			columnName string
			columnType string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return nil, err
		}
		columns[columnName] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	s.columnsMu.Lock()
	if s.columnsCache == nil {
		s.columnsCache = make(map[string]map[string]bool)
	}
	s.columnsCache[name] = columns
	s.columnsMu.Unlock()

	out := make(map[string]bool, len(columns))
	for column := range columns {
		out[column] = true
	}
	return out, nil
}

func quoteSQLiteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func scanDep(rows interface{ Scan(...any) error }) (Dep, error) {
	var dep Dep
	var issueID, dependsOnID, depType sql.NullString
	if err := rows.Scan(&issueID, &dependsOnID, &depType); err != nil {
		return dep, err
	}
	dep.IssueID = issueID.String
	dep.DependsOnID = dependsOnID.String
	dep.Type = depType.String
	if dep.Type == "" {
		dep.Type = "blocks"
	}
	return dep, nil
}

func (s *DoltliteReadStore) queryIssues(query ListQuery, extraWhere string, extraArgs []any, limit int) ([]Bead, error) {
	return s.queryIssuesOrdered(query, extraWhere, extraArgs, limit, "")
}

func (s *DoltliteReadStore) queryIssuesOrdered(query ListQuery, extraWhere string, extraArgs []any, limit int, orderBy string) ([]Bead, error) {
	return s.queryIssuesOrderedInTables(query, doltliteTableSetsForMode(query.TierMode), extraWhere, extraArgs, limit, orderBy)
}

// queryIssuesOrderedInTables runs the query against an explicit list of table
// sets. Callers passing a custom orderBy must pass a single table set: the
// merged path re-sorts only when orderBy is empty.
func (s *DoltliteReadStore) queryIssuesOrderedInTables(query ListQuery, sets []doltliteTableSet, extraWhere string, extraArgs []any, limit int, orderBy string) ([]Bead, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	merged := make([]Bead, 0)
	seen := make(map[string]struct{})
	for _, tables := range sets {
		// A per-table LIMIT of the merged query limit is enough here: the
		// final top-N can never contain more than N rows from any one table.
		tableLimit := limit
		if len(sets) > 1 && !doltliteCanPushTableLimit(query) {
			tableLimit = 0
		}
		rows, err := s.queryIssueTable(query, tables, extraWhere, extraArgs, tableLimit, orderBy)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if _, ok := seen[row.ID]; ok {
				continue
			}
			seen[row.ID] = struct{}{}
			merged = append(merged, row)
		}
	}
	if len(query.Metadata) > 0 {
		merged = filterDoltliteMetadata(merged, query.Metadata)
	}
	merged = filterDoltliteBeforeTimes(merged, query)
	if orderBy == "" {
		sortBeadsForQuery(merged, doltliteSortOrder(query.Sort))
	}
	if limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}
	return merged, nil
}

func doltliteSortOrder(order SortOrder) SortOrder {
	if order == SortCreatedAsc {
		return SortCreatedAsc
	}
	return SortCreatedDesc
}

// doltliteMetadataFilterPredicates narrows metadata queries in SQL without
// relying on SQLite JSON1, which is not available in every embedded build.
// scanBead still applies exact parseMetadata filtering to these candidates.
func doltliteMetadataFilterPredicates(filters map[string]string) ([]string, []any) {
	if len(filters) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(filters))
	for key := range filters {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	where := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)*2)
	for _, key := range keys {
		patterns := doltliteMetadataLikePatterns(key, filters[key])
		clauses := make([]string, 0, len(patterns))
		for _, pattern := range patterns {
			clauses = append(clauses, "i.metadata LIKE ? ESCAPE '\\'")
			args = append(args, pattern)
		}
		where = append(where, "("+strings.Join(clauses, " OR ")+")")
	}
	return where, args
}

func doltliteMetadataEqualsPredicate(alias, key, value string) (string, []any) {
	patterns := doltliteMetadataLikePatterns(key, value)
	clauses := make([]string, 0, len(patterns))
	args := make([]any, 0, len(patterns))
	for _, pattern := range patterns {
		clauses = append(clauses, alias+".metadata LIKE ? ESCAPE '\\'")
		args = append(args, pattern)
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args
}

func doltliteMetadataLikePatterns(key, value string) []string {
	keyJSON, _ := json.Marshal(key)
	valueJSON, _ := json.Marshal(value)
	fragments := []string{
		string(keyJSON) + ":" + string(valueJSON),
		string(keyJSON) + ": " + string(valueJSON),
		string(keyJSON) + " :" + string(valueJSON),
		string(keyJSON) + " : " + string(valueJSON),
	}
	patterns := make([]string, 0, len(fragments))
	seen := make(map[string]struct{}, len(fragments))
	for _, fragment := range fragments {
		pattern := "%" + escapeDoltliteLikePattern(fragment) + "%"
		if _, ok := seen[pattern]; ok {
			continue
		}
		seen[pattern] = struct{}{}
		patterns = append(patterns, pattern)
	}
	return patterns
}

func escapeDoltliteLikePattern(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\\', '%', '_':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func filterDoltliteMetadata(rows []Bead, filters map[string]string) []Bead {
	if len(filters) == 0 || len(rows) == 0 {
		return rows
	}
	out := rows[:0]
	for _, row := range rows {
		if matchesMetadata(row, filters) {
			out = append(out, row)
		}
	}
	return out
}

func (s *DoltliteReadStore) queryIssueTable(query ListQuery, tables doltliteTableSet, extraWhere string, extraArgs []any, limit int, orderBy string) ([]Bead, error) {
	if tables.wisps && !s.tableExists(tables.issues) {
		return nil, nil
	}
	flags := s.storageFlagExprsFor(tables)
	where := []string{}
	args := []any{}
	needParent := true
	tierWhere, skipTable := doltliteTierPredicate(query.TierMode, tables, flags)
	if skipTable {
		return nil, nil
	}
	if tierWhere != "" {
		where = append(where, tierWhere)
	}
	if !query.IncludeClosed && query.Status != "closed" {
		where = append(where, "i.status != 'closed'")
	}
	if query.Status != "" {
		where = append(where, "i.status = ?")
		args = append(args, query.Status)
	}
	if query.Type != "" {
		where = append(where, "i.issue_type = ?")
		args = append(args, query.Type)
	}
	if query.Assignee != "" {
		where = append(where, "i.assignee = ?")
		args = append(args, query.Assignee)
	}
	if len(query.Assignees) > 0 {
		assignees := compactStrings(query.Assignees)
		if len(assignees) == 0 {
			return nil, nil
		}
		placeholders := strings.TrimRight(strings.Repeat("?,", len(assignees)), ",")
		where = append(where, "i.assignee IN ("+placeholders+")")
		for _, assignee := range assignees {
			args = append(args, assignee)
		}
	}
	if query.ParentID != "" {
		where = append(where, s.doltliteDependsOnExpr(tables.deps, "pc")+" = ?")
		args = append(args, query.ParentID)
	}
	if query.Label != "" {
		where = append(where, "EXISTS (SELECT 1 FROM "+tables.labels+" l WHERE l.issue_id = i.id AND l.label = ?)")
		args = append(args, query.Label)
	}
	if len(query.Metadata) > 0 {
		metadataWhere, metadataArgs := doltliteMetadataFilterPredicates(query.Metadata)
		where = append(where, metadataWhere...)
		args = append(args, metadataArgs...)
	}
	if !query.CreatedBefore.IsZero() {
		where = append(where, "julianday(i.created_at) < julianday(?)")
		args = append(args, doltliteSQLiteTime(query.CreatedBefore))
	}
	if !query.UpdatedBefore.IsZero() {
		where = append(where, "julianday(COALESCE(NULLIF(i.updated_at, ''), i.created_at)) < julianday(?)")
		args = append(args, doltliteSQLiteTime(query.UpdatedBefore))
	}
	if extraWhere != "" {
		where = append(where, extraWhere)
		args = append(args, extraArgs...)
	}
	parentColumn := "''"
	parentJoin := ""
	if needParent {
		var err error
		parentColumn, err = s.doltliteDependsOnExprForTable(tables.deps, "pc")
		if err != nil {
			return nil, err
		}
		parentJoin = " LEFT JOIN " + tables.deps + " pc ON pc.issue_id = i.id AND pc.type = 'parent-child'"
	}
	sqlText := `SELECT i.id, COALESCE(i.title, ''), COALESCE(i.status, ''), COALESCE(i.issue_type, ''), i.priority, i.created_at,
		COALESCE(i.updated_at, ''), COALESCE(i.assignee, ''), COALESCE(i.description, ''), COALESCE(i.metadata, '{}'),
		` + parentColumn + `, ` + flags.ephemeral + `, ` + flags.noHistory + `
		FROM ` + tables.issues + ` i` + parentJoin
	if len(where) > 0 {
		sqlText += " WHERE " + strings.Join(where, " AND ")
	}
	// The id tiebreaker matches sortBeadsForQuery's (created_at, id) total
	// order so a SQL LIMIT cuts a deterministic prefix even when rows share
	// a created_at timestamp (#3208).
	if orderBy != "" {
		sqlText += " " + orderBy
	} else if query.Sort == SortCreatedAsc {
		sqlText += " ORDER BY i.created_at ASC, i.id ASC"
	} else {
		sqlText += " ORDER BY i.created_at DESC, i.id DESC"
	}
	if limit > 0 {
		sqlText += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var beads []Bead
	for rows.Next() {
		b, err := scanBead(rows)
		if err != nil {
			return nil, err
		}
		beads = append(beads, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !query.SkipLabels {
		if err := s.hydrateLabels(beads, tables.labels); err != nil {
			return nil, err
		}
	}
	return beads, nil
}

// doltliteStorageFlagExprs holds SQL expressions yielding the per-row
// ephemeral and no_history storage flags for one storage table, accounting
// for snapshots whose schema predates those columns.
type doltliteStorageFlagExprs struct {
	ephemeral string
	noHistory string
	// hasColumns reports whether the table carries at least one storage-flag
	// column, i.e. whether per-row tier classification is possible.
	hasColumns bool
}

// storageFlagExprsFor resolves the storage-flag expressions for tables.
// Legacy snapshots wrote wisps tables without the flag columns; every row
// there is ephemeral, so the wisps fallback is the constant 1 while the
// issues-table fallback is the constant 0 (durable history rows).
func (s *DoltliteReadStore) storageFlagExprsFor(tables doltliteTableSet) doltliteStorageFlagExprs {
	flags := doltliteStorageFlagExprs{ephemeral: "0", noHistory: "0"}
	if tables.wisps {
		flags.ephemeral = "1"
	}
	if s.tableHasColumn(tables.issues, "no_history") {
		flags.noHistory = "COALESCE(i.no_history, 0)"
		flags.hasColumns = true
	}
	if s.tableHasColumn(tables.issues, "ephemeral") {
		flags.ephemeral = "COALESCE(i.ephemeral, 0)"
		if tables.wisps {
			flags.ephemeral = "CASE WHEN " + flags.noHistory + " = 1 THEN 0 ELSE 1 END"
		}
		flags.hasColumns = true
	}
	return flags
}

// doltliteTierPredicate translates query.go's TierMode row filter (Matches)
// into a SQL predicate for one storage table. It returns skipTable=true when
// the table cannot hold rows for the tier at all (a legacy wisps table is
// ephemeral-only, so the durable tier never reads it).
func doltliteTierPredicate(mode TierMode, tables doltliteTableSet, flags doltliteStorageFlagExprs) (string, bool) {
	switch mode {
	case TierWisps:
		if !flags.hasColumns {
			// Legacy wisps rows are all ephemeral; issues-table rows never
			// reach TierWisps because doltliteTableSetsForMode excludes them.
			return "", false
		}
		return "(" + flags.ephemeral + " = 1 OR " + flags.noHistory + " = 1)", false
	case TierBoth:
		return "", false
	default: // TierIssues keeps history and no-history rows, drops ephemeral.
		if tables.wisps && !flags.hasColumns {
			return "", true
		}
		if flags.ephemeral == "0" {
			return "", false
		}
		return flags.ephemeral + " = 0", false
	}
}

// doltliteCanPushTableLimit reports whether a per-table SQL LIMIT cuts an
// exact prefix for a multi-table merge: each table returns its own
// top-limit prefix in the shared (created_at, id) total order, so the merged
// sort+limit is exact unless a Go-side filter (metadata LIKE refinement,
// before-time re-checks) can still drop rows after the SQL cut.
func doltliteCanPushTableLimit(query ListQuery) bool {
	return len(query.Metadata) == 0 && query.CreatedBefore.IsZero() && query.UpdatedBefore.IsZero()
}

func doltliteSQLiteTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05.999999999-07:00")
}

func filterDoltliteBeforeTimes(rows []Bead, query ListQuery) []Bead {
	if len(rows) == 0 || (query.CreatedBefore.IsZero() && query.UpdatedBefore.IsZero()) {
		return rows
	}
	out := rows[:0]
	for _, row := range rows {
		if !query.CreatedBefore.IsZero() && !row.CreatedAt.Before(query.CreatedBefore) {
			continue
		}
		if !query.UpdatedBefore.IsZero() && !beadUpdatedReferenceTime(row).Before(query.UpdatedBefore) {
			continue
		}
		out = append(out, row)
	}
	return out
}

func scanBead(rows interface{ Scan(...any) error }) (Bead, error) {
	var (
		b           Bead
		priority    sql.NullInt64
		createdRaw  any
		updatedRaw  any
		metadataRaw string
		ephemeral   int64
		noHistory   int64
	)
	if columns, ok := rows.(interface{ Columns() ([]string, error) }); ok {
		names, err := columns.Columns()
		if err != nil {
			return b, err
		}
		if len(names) <= 11 {
			if err := rows.Scan(&b.ID, &b.Title, &b.Status, &b.Type, &priority, &createdRaw, &updatedRaw, &b.Assignee, &b.Description, &metadataRaw, &b.ParentID); err != nil {
				return b, err
			}
		} else if err := rows.Scan(&b.ID, &b.Title, &b.Status, &b.Type, &priority, &createdRaw, &updatedRaw, &b.Assignee, &b.Description, &metadataRaw, &b.ParentID, &ephemeral, &noHistory); err != nil {
			return b, err
		}
	} else {
		if err := rows.Scan(&b.ID, &b.Title, &b.Status, &b.Type, &priority, &createdRaw, &updatedRaw, &b.Assignee, &b.Description, &metadataRaw, &b.ParentID, &ephemeral, &noHistory); err != nil {
			return b, err
		}
	}
	if priority.Valid {
		p := int(priority.Int64)
		b.Priority = &p
	}
	b.Status = mapBdStatus(b.Status)
	b.CreatedAt = parseDBTime(createdRaw).Truncate(time.Second)
	b.UpdatedAt = parseDBTime(updatedRaw).Truncate(time.Second)
	b.Metadata = parseMetadata(metadataRaw)
	b.Ephemeral = ephemeral != 0
	b.NoHistory = noHistory != 0
	if b.From == "" {
		b.From = b.Metadata["from"]
	}
	return b, nil
}

func parseDBTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		return parseTimeString(t)
	case []byte:
		return parseTimeString(string(t))
	default:
		return time.Time{}
	}
}

func parseTimeString(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999 -0700 MST", // legacy time.Time.String() write format
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func parseMetadata(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return nil
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil
	}
	out := make(map[string]string, len(decoded))
	for k, v := range decoded {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			out[k] = s
		} else {
			out[k] = strings.TrimSpace(string(v))
		}
	}
	return out
}

func (s *DoltliteReadStore) tableExists(name string) bool {
	var found string
	err := s.db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&found)
	return err == nil
}

// tableHasColumn reports whether the table's schema includes the named
// column. Snapshot schemas vary by writer generation (the storage-flag
// columns arrived with the upstream wisps/no-history migrations), so read
// paths probe before referencing them.
func (s *DoltliteReadStore) tableHasColumn(table, column string) bool {
	return s.columnExists(table, column)
}

func (s *DoltliteReadStore) doltliteDependsOnExpr(table, alias string) string {
	expr, err := s.doltliteDependsOnExprForTable(table, alias)
	if err != nil {
		return "''"
	}
	return expr
}

func (s *DoltliteReadStore) columnExists(table, column string) bool {
	columns, err := s.tableColumns(table)
	if err != nil {
		return false
	}
	return columns[column]
}

func (s *DoltliteReadStore) columnsForTable(table string) map[string]bool {
	columns, err := s.tableColumns(table)
	if err != nil {
		return map[string]bool{}
	}
	return columns
}

func (s *DoltliteReadStore) warmSchemaCache() {
	// libdoltlite is sensitive to late schema PRAGMA probes while another
	// handle has the file open for writes. The schema is stable for a store
	// lifetime, so populate the known tables before normal read/write traffic.
	for _, table := range []string{
		doltliteIssueTables.issues,
		doltliteIssueTables.labels,
		doltliteIssueTables.deps,
		doltliteWispTables.issues,
		doltliteWispTables.labels,
		doltliteWispTables.deps,
	} {
		_ = s.columnsForTable(table)
	}
}

func doltliteQuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (s *DoltliteReadStore) hydrateLabels(beads []Bead, labelTable string) error {
	if len(beads) == 0 {
		return nil
	}
	byID := make(map[string]*Bead, len(beads))
	args := make([]any, 0, len(beads))
	for i := range beads {
		byID[beads[i].ID] = &beads[i]
		args = append(args, beads[i].ID)
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(args)), ",")
	rows, err := s.db.Query(`SELECT issue_id, label FROM `+labelTable+` WHERE issue_id IN (`+placeholders+`)`, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, label string
		if err := rows.Scan(&id, &label); err != nil {
			return err
		}
		if b := byID[id]; b != nil {
			b.Labels = append(b.Labels, label)
		}
	}
	for i := range beads {
		sort.Strings(beads[i].Labels)
	}
	return rows.Err()
}
