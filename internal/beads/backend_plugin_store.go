package beads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const gascityBackendPluginProtocol = "gascity.backend.v1alpha1"

type backendPluginMetadata struct {
	Backend                    string   `json:"backend"`
	Database                   string   `json:"database"`
	DoltDatabase               string   `json:"dolt_database"`
	BackendPluginCommand       string   `json:"backend_plugin_command"`
	BackendPluginArgs          []string `json:"backend_plugin_args"`
	GasCityBackendCommand      string   `json:"gascity_backend_command"`
	GasCityBackendArgs         []string `json:"gascity_backend_args"`
	GasCityFastpathCommand     string   `json:"gascity_fastpath_command"`
	GasCityFastpathArgs        []string `json:"gascity_fastpath_args"`
	GasCityFastpathPlugin      string   `json:"gc_fastpath_plugin_command"`
	GasCityFastpathPluginArgs  []string `json:"gc_fastpath_plugin_args"`
	DoltLiteFastpathCommand    string   `json:"doltlite_fastpath_command"`
	DoltLiteFastpathPluginArgs []string `json:"doltlite_fastpath_args"`
}

type backendPluginHello struct {
	Protocol     string                    `json:"protocol"`
	Backend      string                    `json:"backend"`
	Capabilities backendPluginCapabilities `json:"capabilities"`
}

type backendPluginCapabilities struct {
	GetIssue         bool `json:"get_issue"`
	SearchIssues     bool `json:"search_issues"`
	ReadyWork        bool `json:"ready_work"`
	ListWisps        bool `json:"list_wisps"`
	CountIssues      bool `json:"count_issues"`
	Labels           bool `json:"labels"`
	Dependencies     bool `json:"dependencies"`
	StorageCreate    bool `json:"storage_create"`
	ConditionalClaim bool `json:"conditional_claim"`
	BatchDeps        bool `json:"batch_deps"`
	WriteOperations  bool `json:"write_operations"`
}

type backendPluginRequest struct {
	ID     string          `json:"id,omitempty"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type backendPluginResponse struct {
	ID     string          `json:"id,omitempty"`
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type backendPluginClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	enc    *json.Encoder
	dec    *json.Decoder

	mu      sync.Mutex
	nextID  atomic.Uint64
	closed  atomic.Bool
	timeout time.Duration
	hello   backendPluginHello
}

// BackendPluginStore implements Store by talking to a plugin-owned Gas City
// backend process. gc stays pure Go; backend-specific database access lives in
// the plugin process.
type BackendPluginStore struct {
	backing   *BdStore
	client    *backendPluginClient
	sessionID string
	scopeRoot string
	database  string
}

// NewBackendPluginStore opens a Gas City bead store over the configured
// backend plugin protocol.
func NewBackendPluginStore(scopeRoot string, backing *BdStore) (*BackendPluginStore, error) {
	scopeRoot = strings.TrimSpace(scopeRoot)
	if scopeRoot == "" {
		return nil, errors.New("backend plugin store: scope root is required")
	}
	meta, err := readBackendPluginMetadata(scopeRoot)
	if err != nil {
		return nil, err
	}
	command, args, err := resolveBackendPluginCommand(meta)
	if err != nil {
		return nil, err
	}
	client, err := startBackendPluginClient(command, args)
	if err != nil {
		return nil, err
	}
	database := strings.TrimSpace(meta.DoltDatabase)
	if database == "" || database == "doltlite" {
		database = strings.TrimSpace(meta.Database)
	}
	if database == "" || database == "doltlite" {
		database = "hq"
	}
	var opened struct {
		SessionID string `json:"session_id"`
	}
	if err := client.request("open", map[string]any{
		"beads_dir": filepath.Join(scopeRoot, ".beads"),
		"database":  database,
		"branch":    "main",
	}, &opened); err != nil {
		_ = client.Close()
		return nil, err
	}
	if strings.TrimSpace(opened.SessionID) == "" {
		_ = client.Close()
		return nil, errors.New("backend plugin store: plugin returned empty session id")
	}
	return &BackendPluginStore{
		backing:   backing,
		client:    client,
		sessionID: opened.SessionID,
		scopeRoot: scopeRoot,
		database:  database,
	}, nil
}

func readBackendPluginMetadata(scopeRoot string) (backendPluginMetadata, error) {
	var meta backendPluginMetadata
	data, err := os.ReadFile(filepath.Join(scopeRoot, ".beads", "metadata.json"))
	if err != nil {
		return meta, err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return meta, fmt.Errorf("backend plugin store: parsing metadata.json: %w", err)
	}
	return meta, nil
}

func resolveBackendPluginCommand(meta backendPluginMetadata) (string, []string, error) {
	for _, candidate := range []struct {
		command string
		args    []string
	}{
		{meta.GasCityBackendCommand, meta.GasCityBackendArgs},
		{meta.GasCityFastpathCommand, meta.GasCityFastpathArgs},
		{meta.GasCityFastpathPlugin, meta.GasCityFastpathPluginArgs},
		{meta.DoltLiteFastpathCommand, meta.DoltLiteFastpathPluginArgs},
	} {
		if strings.TrimSpace(candidate.command) != "" {
			return candidate.command, defaultBackendPluginArgs(candidate.args), nil
		}
	}
	if strings.TrimSpace(meta.BackendPluginCommand) == "" {
		return "", nil, errors.New("backend plugin store: metadata.json missing gascity_backend_command")
	}
	command := filepath.Join(filepath.Dir(meta.BackendPluginCommand), "gc-doltlite-fastpath")
	return command, []string{"serve"}, nil
}

func defaultBackendPluginArgs(args []string) []string {
	if len(args) == 0 {
		return []string{"serve"}
	}
	out := append([]string(nil), args...)
	return out
}

func startBackendPluginClient(command string, args []string) (*backendPluginClient, error) {
	cmd := exec.Command(command, args...) // #nosec G204 - command comes from trusted project metadata.
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("backend plugin store stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("backend plugin store stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start backend plugin %q: %w", command, err)
	}
	client := &backendPluginClient{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		enc:     json.NewEncoder(stdin),
		dec:     json.NewDecoder(stdout),
		timeout: 30 * time.Second,
	}
	if err := client.readHello(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func (c *backendPluginClient) readHello() error {
	var resp backendPluginResponse
	if err := c.dec.Decode(&resp); err != nil {
		return fmt.Errorf("read backend plugin hello: %w", err)
	}
	if !resp.OK {
		return responseError("hello", resp)
	}
	if err := json.Unmarshal(resp.Result, &c.hello); err != nil {
		return fmt.Errorf("decode backend plugin hello: %w", err)
	}
	if c.hello.Protocol != gascityBackendPluginProtocol {
		return fmt.Errorf("backend plugin protocol %q, want %q", c.hello.Protocol, gascityBackendPluginProtocol)
	}
	return nil
}

func (c *backendPluginClient) request(method string, params any, out any) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal %s request: %w", method, err)
	}
	id := fmt.Sprintf("%d", c.nextID.Add(1))
	req := backendPluginRequest{ID: id, Method: method, Params: data}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed.Load() {
		return errors.New("backend plugin client is closed")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.enc.Encode(req); err != nil {
		return fmt.Errorf("send %s request: %w", method, err)
	}
	var resp backendPluginResponse
	if err := c.dec.Decode(&resp); err != nil {
		return fmt.Errorf("read %s response: %w", method, err)
	}
	if resp.ID != "" && resp.ID != id {
		return fmt.Errorf("backend plugin response id %q does not match request id %q", resp.ID, id)
	}
	if !resp.OK {
		return responseError(method, resp)
	}
	if out == nil {
		return nil
	}
	if len(resp.Result) == 0 {
		resp.Result = []byte("{}")
	}
	if err := json.Unmarshal(resp.Result, out); err != nil {
		return fmt.Errorf("decode %s response: %w", method, err)
	}
	return nil
}

func responseError(method string, resp backendPluginResponse) error {
	if resp.Error == nil {
		return fmt.Errorf("%s failed", method)
	}
	if resp.Error.Code == "not_found" {
		return fmt.Errorf("%s failed: %s: %w", method, resp.Error.Message, ErrNotFound)
	}
	return fmt.Errorf("%s failed: %s: %s", method, resp.Error.Code, resp.Error.Message)
}

func (c *backendPluginClient) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	var err error
	if c.stdin != nil {
		err = errors.Join(err, c.stdin.Close())
	}
	if c.stdout != nil {
		err = errors.Join(err, c.stdout.Close())
	}
	if c.cmd != nil {
		err = errors.Join(err, c.cmd.Wait())
	}
	return err
}

func (s *BackendPluginStore) CloseStore() error {
	var err error
	if s.client != nil && s.sessionID != "" {
		err = errors.Join(err, s.client.request("close", map[string]string{"session_id": s.sessionID}, nil))
	}
	if s.client != nil {
		err = errors.Join(err, s.client.Close())
	}
	return err
}

func (s *BackendPluginStore) Ping() error {
	if s.client == nil {
		return errors.New("backend plugin store is closed")
	}
	return nil
}

func (s *BackendPluginStore) Create(b Bead) (Bead, error) {
	if !s.client.hello.Capabilities.WriteOperations {
		return Bead{}, errors.New("backend plugin store: write operations unavailable")
	}
	issue := beadToPluginIssue(b)
	var created backendPluginIssue
	if err := s.client.request("create_issue", map[string]any{
		"session_id": s.sessionID,
		"issue":      issue,
		"actor":      "gc",
		"commit":     true,
		"message":    "gc create bead",
	}, &created); err != nil {
		return Bead{}, err
	}
	out := pluginIssueToBead(created)
	for _, dep := range createDependenciesForBead(out.ID, b) {
		if err := s.addPluginDependency(dep); err != nil {
			return Bead{}, err
		}
	}
	if len(b.Needs) > 0 || b.ParentID != "" {
		refreshed, err := s.Get(out.ID)
		if err == nil {
			out = refreshed
		}
	}
	return out, nil
}

func (s *BackendPluginStore) CreateWithStorage(b Bead, storage StorageClass) (Bead, error) {
	switch storage {
	case StorageNoHistory:
		b.NoHistory = true
		b.Ephemeral = false
	case StorageEphemeral:
		b.Ephemeral = true
		b.NoHistory = false
	case StorageHistory:
		b.Ephemeral = false
		b.NoHistory = false
	}
	return s.Create(b)
}

func (s *BackendPluginStore) Get(id string) (Bead, error) {
	var issue backendPluginIssue
	if err := s.client.request("get_issue", map[string]string{"session_id": s.sessionID, "id": id}, &issue); err != nil {
		return Bead{}, err
	}
	return pluginIssueToBead(issue), nil
}

func (s *BackendPluginStore) Update(id string, opts UpdateOpts) error {
	if !s.client.hello.Capabilities.WriteOperations {
		return errors.New("backend plugin store: write operations unavailable")
	}
	updates := pluginUpdatesFromOpts(opts)
	if parentID, ok := updateParentID(opts); ok {
		if err := s.reparent(id, parentID); err != nil {
			return err
		}
	}
	if len(updates) == 0 {
		return nil
	}
	var ignored backendPluginIssue
	return s.client.request("update_issue", map[string]any{
		"session_id": s.sessionID,
		"id":         id,
		"updates":    updates,
		"actor":      "gc",
		"commit":     true,
		"message":    "gc update bead " + id,
	}, &ignored)
}

func (s *BackendPluginStore) Close(id string) error {
	return s.client.request("close_issue", map[string]any{
		"session_id": s.sessionID,
		"id":         id,
		"actor":      "gc",
		"reason":     "gc close",
	}, nil)
}

func (s *BackendPluginStore) CloseAll(ids []string, metadata map[string]string) (int, error) {
	closed := 0
	for _, id := range ids {
		if len(metadata) > 0 {
			if err := s.SetMetadataBatch(id, metadata); err != nil {
				return closed, err
			}
		}
		bead, err := s.Get(id)
		if err != nil {
			return closed, err
		}
		if bead.Status == "closed" {
			continue
		}
		if err := s.Close(id); err != nil {
			return closed, err
		}
		closed++
	}
	return closed, nil
}

func (s *BackendPluginStore) Reopen(id string) error {
	return s.client.request("reopen_issue", map[string]any{
		"session_id": s.sessionID,
		"id":         id,
		"actor":      "gc",
		"reason":     "gc reopen",
	}, nil)
}

func (s *BackendPluginStore) Delete(id string) error {
	return s.client.request("delete_issue", map[string]any{
		"session_id": s.sessionID,
		"id":         id,
		"actor":      "gc",
	}, nil)
}

func (s *BackendPluginStore) ReleaseIfCurrent(id, expectedAssignee string) (bool, error) {
	current, err := s.Get(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if current.Status != "in_progress" || current.Assignee != expectedAssignee {
		return false, nil
	}
	status := "open"
	assignee := ""
	if err := s.Update(id, UpdateOpts{Status: &status, Assignee: &assignee}); err != nil {
		return false, err
	}
	return true, nil
}

func (s *BackendPluginStore) List(query ListQuery) ([]Bead, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	if !query.HasFilter() && !query.AllowScan {
		return nil, fmt.Errorf("bd list: %w", ErrQueryRequiresScan)
	}
	if len(query.Assignees) > 1 {
		return s.listByAssignees(query)
	}
	var issues []backendPluginIssue
	if query.TierMode == TierWisps {
		if err := s.client.request("list_wisps", map[string]any{
			"session_id": s.sessionID,
			"filter":     wispFilterFromListQuery(query),
		}, &issues); err != nil {
			return nil, err
		}
		return pluginIssuesToBeads(issues, query), nil
	}
	if err := s.client.request("search_issues", map[string]any{
		"session_id": s.sessionID,
		"filter":     issueFilterFromListQuery(query),
	}, &issues); err != nil {
		return nil, err
	}
	return pluginIssuesToBeads(issues, query), nil
}

func (s *BackendPluginStore) ListOpen(status ...string) ([]Bead, error) {
	query := ListQuery{AllowScan: true}
	if len(status) > 0 {
		query.Status = strings.TrimSpace(status[0])
	}
	return s.List(query)
}

func (s *BackendPluginStore) Ready(query ...ReadyQuery) ([]Bead, error) {
	rq := readyQueryFromArgs(query)
	if len(rq.Assignees) > 1 {
		return s.readyByAssignees(rq)
	}
	var issues []backendPluginIssue
	if err := s.client.request("ready_work", map[string]any{
		"session_id": s.sessionID,
		"filter":     workFilterFromReadyQuery(rq),
	}, &issues); err != nil {
		return nil, err
	}
	beads := pluginIssuesToBeads(issues, ListQuery{TierMode: rq.TierMode})
	if rq.Limit > 0 && len(beads) > rq.Limit {
		beads = beads[:rq.Limit]
	}
	return beads, nil
}

func (s *BackendPluginStore) listByAssignees(query ListQuery) ([]Bead, error) {
	seen := map[string]bool{}
	var merged []Bead
	for _, assignee := range query.Assignees {
		assignee = strings.TrimSpace(assignee)
		if assignee == "" {
			continue
		}
		q := query
		q.Assignee = assignee
		q.Assignees = nil
		rows, err := s.List(q)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if seen[row.ID] {
				continue
			}
			seen[row.ID] = true
			merged = append(merged, row)
		}
	}
	if query.Limit > 0 && len(merged) > query.Limit {
		merged = merged[:query.Limit]
	}
	return merged, nil
}

func (s *BackendPluginStore) readyByAssignees(query ReadyQuery) ([]Bead, error) {
	seen := map[string]bool{}
	var merged []Bead
	for _, assignee := range query.Assignees {
		assignee = strings.TrimSpace(assignee)
		if assignee == "" {
			continue
		}
		q := query
		q.Assignee = assignee
		q.Assignees = nil
		rows, err := s.Ready(q)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if seen[row.ID] {
				continue
			}
			seen[row.ID] = true
			merged = append(merged, row)
		}
	}
	if query.Limit > 0 && len(merged) > query.Limit {
		merged = merged[:query.Limit]
	}
	return merged, nil
}

func (s *BackendPluginStore) Children(parentID string, opts ...QueryOpt) ([]Bead, error) {
	return s.List(ListQuery{
		ParentID:      parentID,
		IncludeClosed: HasOpt(opts, IncludeClosed),
		AllowScan:     true,
		Sort:          SortCreatedAsc,
		TierMode:      TierModeFromOpts(opts),
	})
}

func (s *BackendPluginStore) ListByLabel(label string, limit int, opts ...QueryOpt) ([]Bead, error) {
	return s.List(ListQuery{
		Label:         label,
		Limit:         limit,
		IncludeClosed: HasOpt(opts, IncludeClosed),
		Sort:          SortCreatedDesc,
		TierMode:      TierModeFromOpts(opts),
	})
}

func (s *BackendPluginStore) ListByAssignee(assignee, status string, limit int) ([]Bead, error) {
	return s.List(ListQuery{Assignee: assignee, Status: status, Limit: limit})
}

func (s *BackendPluginStore) ListByMetadata(filters map[string]string, limit int, opts ...QueryOpt) ([]Bead, error) {
	return s.List(ListQuery{
		Metadata:      filters,
		Limit:         limit,
		IncludeClosed: HasOpt(opts, IncludeClosed),
		Sort:          SortCreatedDesc,
		TierMode:      TierModeFromOpts(opts),
	})
}

func (s *BackendPluginStore) Count(ctx context.Context, query ListQuery, excludeTypes ...string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if err := query.Validate(); err != nil {
		return 0, err
	}
	if !query.HasFilter() && !query.AllowScan {
		return 0, fmt.Errorf("bd count: %w", ErrQueryRequiresScan)
	}
	if query.Limit > 0 {
		return 0, fmt.Errorf("bd count: %w", ErrCountUnsupported)
	}
	filter := issueFilterFromListQuery(query)
	filter.ExcludeTypes = append(filter.ExcludeTypes, compactPluginStrings(excludeTypes)...)
	var result struct {
		Count int `json:"count"`
	}
	if err := s.client.request("count_issues", map[string]any{
		"session_id": s.sessionID,
		"filter":     filter,
	}, &result); err != nil {
		return 0, err
	}
	return result.Count, nil
}

func (s *BackendPluginStore) SetMetadata(id, key, value string) error {
	return s.client.request("slot_set", map[string]string{
		"session_id": s.sessionID,
		"issue_id":   id,
		"key":        key,
		"value":      value,
		"actor":      "gc",
	}, nil)
}

func (s *BackendPluginStore) SetMetadataBatch(id string, kvs map[string]string) error {
	keys := make([]string, 0, len(kvs))
	for key := range kvs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := s.SetMetadata(id, key, kvs[key]); err != nil {
			return err
		}
	}
	return nil
}

func (s *BackendPluginStore) Tx(_ string, fn func(tx Tx) error) error {
	return runSequentialTx(s, fn)
}

func (s *BackendPluginStore) DepAdd(issueID, dependsOnID, depType string) error {
	return s.addPluginDependency(backendPluginDependency{IssueID: issueID, DependsOnID: dependsOnID, Type: defaultDepType(depType)})
}

func (s *BackendPluginStore) DepRemove(issueID, dependsOnID string) error {
	return s.client.request("remove_dependency", map[string]string{
		"session_id":    s.sessionID,
		"issue_id":      issueID,
		"depends_on_id": dependsOnID,
		"actor":         "gc",
	}, nil)
}

func (s *BackendPluginStore) DepList(id, direction string) ([]Dep, error) {
	method := "get_dependencies"
	if direction == "up" {
		method = "get_dependents"
	}
	var deps []backendPluginDependency
	if err := s.client.request(method, map[string]string{
		"session_id": s.sessionID,
		"issue_id":   id,
	}, &deps); err != nil {
		return nil, err
	}
	out := make([]Dep, 0, len(deps))
	for _, dep := range deps {
		out = append(out, Dep{IssueID: dep.IssueID, DependsOnID: dep.DependsOnID, Type: dep.Type})
	}
	return out, nil
}

func (s *BackendPluginStore) DepListBatch(ids []string) (map[string][]Dep, error) {
	out := make(map[string][]Dep, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		deps, err := s.DepList(id, "down")
		if err != nil {
			return nil, err
		}
		out[id] = deps
	}
	return out, nil
}

func (s *BackendPluginStore) addPluginDependency(dep backendPluginDependency) error {
	return s.client.request("add_dependency", map[string]any{
		"session_id": s.sessionID,
		"dependency": dep,
		"actor":      "gc",
	}, nil)
}

func (s *BackendPluginStore) reparent(id, parentID string) error {
	deps, err := s.DepList(id, "down")
	if err != nil {
		return err
	}
	for _, dep := range deps {
		if dep.Type == "parent-child" && dep.DependsOnID != parentID {
			if err := s.DepRemove(id, dep.DependsOnID); err != nil {
				return err
			}
		}
	}
	if strings.TrimSpace(parentID) == "" {
		return nil
	}
	for _, dep := range deps {
		if dep.Type == "parent-child" && dep.DependsOnID == parentID {
			return nil
		}
	}
	return s.DepAdd(id, parentID, "parent-child")
}

func updateParentID(opts UpdateOpts) (string, bool) {
	if opts.ParentID == nil {
		return "", false
	}
	return strings.TrimSpace(*opts.ParentID), true
}

type backendPluginIssue struct {
	ID          string                    `json:"id"`
	Title       string                    `json:"title"`
	Description string                    `json:"description,omitempty"`
	Status      string                    `json:"status,omitempty"`
	Priority    int                       `json:"priority"`
	IssueType   string                    `json:"issue_type,omitempty"`
	Assignee    string                    `json:"assignee,omitempty"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
	DeferUntil  *time.Time                `json:"defer_until,omitempty"`
	ExternalRef *string                   `json:"external_ref,omitempty"`
	Metadata    json.RawMessage           `json:"metadata,omitempty"`
	Labels      []string                  `json:"labels,omitempty"`
	Deps        []backendPluginDependency `json:"dependencies,omitempty"`
	Sender      string                    `json:"sender,omitempty"`
	Ephemeral   bool                      `json:"ephemeral,omitempty"`
	NoHistory   bool                      `json:"no_history,omitempty"`
}

type backendPluginDependency struct {
	IssueID     string    `json:"issue_id"`
	DependsOnID string    `json:"depends_on_id"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	CreatedBy   string    `json:"created_by,omitempty"`
	Metadata    string    `json:"metadata,omitempty"`
	ThreadID    string    `json:"thread_id,omitempty"`
}

func beadToPluginIssue(b Bead) backendPluginIssue {
	priority := 2
	if b.Priority != nil {
		priority = *b.Priority
	}
	metadata := map[string]string{}
	for k, v := range b.Metadata {
		metadata[k] = v
	}
	if b.From != "" && metadata["from"] == "" {
		metadata["from"] = b.From
	}
	rawMetadata, _ := json.Marshal(metadata)
	return backendPluginIssue{
		ID:          b.ID,
		Title:       b.Title,
		Description: b.Description,
		Status:      b.Status,
		Priority:    priority,
		IssueType:   defaultIssueType(b.Type),
		Assignee:    b.Assignee,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
		DeferUntil:  b.DeferUntil,
		Metadata:    rawMetadata,
		Labels:      compactPluginStrings(b.Labels),
		Sender:      b.From,
		Ephemeral:   b.Ephemeral,
		NoHistory:   b.NoHistory,
	}
}

func pluginIssueToBead(issue backendPluginIssue) Bead {
	priority := issue.Priority
	if priority == 0 {
		priority = 2
	}
	metadata := map[string]string{}
	if len(issue.Metadata) > 0 {
		_ = json.Unmarshal(issue.Metadata, &metadata)
	}
	parentID := ""
	deps := make([]Dep, 0, len(issue.Deps))
	for _, dep := range issue.Deps {
		deps = append(deps, Dep{IssueID: dep.IssueID, DependsOnID: dep.DependsOnID, Type: dep.Type})
		if dep.Type == "parent-child" && parentID == "" {
			parentID = dep.DependsOnID
		}
	}
	from := issue.Sender
	if from == "" {
		from = metadata["from"]
	}
	return Bead{
		ID:           issue.ID,
		Title:        issue.Title,
		Status:       defaultStatus(issue.Status),
		Type:         defaultIssueType(issue.IssueType),
		Priority:     &priority,
		CreatedAt:    issue.CreatedAt,
		UpdatedAt:    issue.UpdatedAt,
		Assignee:     issue.Assignee,
		From:         from,
		ParentID:     parentID,
		Ref:          derefString(issue.ExternalRef),
		Description:  issue.Description,
		Labels:       compactPluginStrings(issue.Labels),
		Metadata:     metadata,
		Dependencies: deps,
		Ephemeral:    issue.Ephemeral,
		NoHistory:    issue.NoHistory,
		DeferUntil:   issue.DeferUntil,
	}
}

func pluginIssuesToBeads(issues []backendPluginIssue, query ListQuery) []Bead {
	out := make([]Bead, 0, len(issues))
	for _, issue := range issues {
		bead := pluginIssueToBead(issue)
		if !query.IncludeClosed && bead.Status == "closed" {
			continue
		}
		if query.TierMode == TierWisps && !bead.Ephemeral && !bead.NoHistory {
			continue
		}
		if query.TierMode == TierIssues && bead.Ephemeral {
			continue
		}
		out = append(out, bead)
	}
	return out
}

func createDependenciesForBead(id string, b Bead) []backendPluginDependency {
	var deps []backendPluginDependency
	if parentID := strings.TrimSpace(b.ParentID); parentID != "" {
		deps = append(deps, backendPluginDependency{IssueID: id, DependsOnID: parentID, Type: "parent-child"})
	}
	for _, need := range b.Needs {
		depType, target := parseNeedDependency(need)
		if target != "" {
			deps = append(deps, backendPluginDependency{IssueID: id, DependsOnID: target, Type: depType})
		}
	}
	for _, dep := range b.Dependencies {
		if dep.IssueID == "" {
			dep.IssueID = id
		}
		if dep.DependsOnID != "" {
			deps = append(deps, backendPluginDependency{IssueID: dep.IssueID, DependsOnID: dep.DependsOnID, Type: defaultDepType(dep.Type)})
		}
	}
	return deps
}

func parseNeedDependency(need string) (string, string) {
	need = strings.TrimSpace(need)
	if need == "" {
		return "", ""
	}
	depType, target, ok := strings.Cut(need, ":")
	if !ok {
		return "blocks", need
	}
	return defaultDepType(depType), strings.TrimSpace(target)
}

func pluginUpdatesFromOpts(opts UpdateOpts) map[string]any {
	updates := map[string]any{}
	if opts.Title != nil {
		updates["title"] = *opts.Title
	}
	if opts.Status != nil {
		updates["status"] = *opts.Status
	}
	if opts.Type != nil {
		updates["issue_type"] = *opts.Type
	}
	if opts.Priority != nil {
		updates["priority"] = *opts.Priority
	}
	if opts.Description != nil {
		updates["description"] = *opts.Description
	}
	if opts.Assignee != nil {
		updates["assignee"] = *opts.Assignee
	}
	if len(opts.Metadata) > 0 {
		updates["metadata"] = opts.Metadata
	}
	return updates
}

type backendPluginIssueFilter struct {
	Status         *string
	Statuses       []string
	Priority       *int
	IssueType      *string
	Assignee       *string
	Labels         []string
	IDs            []string
	Limit          int
	CreatedBefore  *time.Time
	UpdatedBefore  *time.Time
	Ephemeral      *bool
	ParentID       *string
	ExcludeStatus  []string
	ExcludeTypes   []string
	MetadataFields map[string]string
	SkipLabels     bool
	SortBy         string
	SortDesc       bool
}

type backendPluginWorkFilter struct {
	Status           string
	Assignee         *string
	Labels           []string
	Limit            int
	IncludeEphemeral bool
}

type backendPluginWispFilter struct {
	Type          *string
	Status        *string
	UpdatedBefore *time.Time
	IncludeClosed bool
	Limit         int
}

func issueFilterFromListQuery(q ListQuery) backendPluginIssueFilter {
	filter := backendPluginIssueFilter{
		Labels:         compactPluginStrings([]string{q.Label}),
		MetadataFields: q.Metadata,
		Limit:          q.Limit,
		CreatedBefore:  timePtr(q.CreatedBefore),
		UpdatedBefore:  timePtr(q.UpdatedBefore),
		SkipLabels:     q.SkipLabels,
	}
	if q.Status != "" {
		filter.Status = stringPtr(q.Status)
	} else if !q.IncludeClosed {
		filter.ExcludeStatus = []string{"closed"}
	}
	if q.Type != "" {
		filter.IssueType = stringPtr(q.Type)
	}
	if q.Assignee != "" {
		filter.Assignee = stringPtr(q.Assignee)
	}
	if len(q.Assignees) > 0 {
		filter.Assignee = stringPtr(q.Assignees[0])
	}
	if q.ParentID != "" {
		filter.ParentID = stringPtr(q.ParentID)
	}
	if q.TierMode == TierWisps {
		filter.Ephemeral = boolPtr(true)
	} else if q.TierMode == TierIssues {
		filter.Ephemeral = boolPtr(false)
	}
	if q.Sort == SortCreatedDesc {
		filter.SortBy = "created"
		filter.SortDesc = true
	} else if q.Sort == SortCreatedAsc {
		filter.SortBy = "created"
	}
	return filter
}

func wispFilterFromListQuery(q ListQuery) backendPluginWispFilter {
	filter := backendPluginWispFilter{IncludeClosed: q.IncludeClosed, Limit: q.Limit, UpdatedBefore: timePtr(q.UpdatedBefore)}
	if q.Status != "" {
		filter.Status = stringPtr(q.Status)
	}
	if q.Type != "" {
		filter.Type = stringPtr(q.Type)
	}
	return filter
}

func workFilterFromReadyQuery(q ReadyQuery) backendPluginWorkFilter {
	filter := backendPluginWorkFilter{Status: "open", Limit: q.Limit}
	if q.Assignee != "" {
		filter.Assignee = stringPtr(q.Assignee)
	} else if len(q.Assignees) > 0 {
		filter.Assignee = stringPtr(q.Assignees[0])
	}
	filter.IncludeEphemeral = q.TierMode == TierWisps || q.TierMode == TierBoth
	return filter
}

func defaultStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "open"
	}
	return status
}

func defaultIssueType(issueType string) string {
	issueType = strings.TrimSpace(issueType)
	if issueType == "" {
		return "task"
	}
	return issueType
}

func defaultDepType(depType string) string {
	depType = strings.TrimSpace(depType)
	if depType == "" {
		return "blocks"
	}
	return depType
}

func compactPluginStrings(values []string) []string {
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

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func stringPtr(v string) *string { return &v }
func boolPtr(v bool) *bool       { return &v }

func timePtr(v time.Time) *time.Time {
	if v.IsZero() {
		return nil
	}
	return &v
}
