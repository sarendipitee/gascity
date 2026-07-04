//go:build gascity_doltlite_lib

package beads

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDoltliteReadStoreListsSessionBeads(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	rows, err := store.List(ListQuery{
		Label: "gc:session",
		Sort:  SortCreatedDesc,
	})
	if err != nil {
		t.Fatalf("List session beads: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("session rows = %d, want 1", len(rows))
	}
	got := rows[0]
	if got.ID != "gc-session" || got.Type != "session" || got.Metadata["session_name"] != "session-1" {
		t.Fatalf("session bead = %#v", got)
	}
	if !slices.Contains(got.Labels, "gc:session") {
		t.Fatalf("labels = %v, missing gc:session", got.Labels)
	}
}

func TestDoltliteReadStoreOpensExistingDoltliteDBWithStaleDoltMetadata(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(filepath.Join(beadsDir, "doltlite"), 0o755); err != nil {
		t.Fatalf("mkdir doltlite dir: %v", err)
	}
	meta := []byte(`{"backend":"dolt","database":"dolt","dolt_database":"hq","dolt_mode":"server"}`)
	if err := os.WriteFile(filepath.Join(beadsDir, "metadata.json"), meta, 0o600); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	db, err := sql.Open(doltliteSQLDriverName, filepath.Join(beadsDir, "doltlite", "hq.db")+"?_busy_timeout=10000")
	if err != nil {
		t.Fatalf("open doltlite fixture db: %v", err)
	}
	createTestDoltliteSchema(t, db)
	if err := db.Close(); err != nil {
		t.Fatalf("close fixture db: %v", err)
	}

	backing := NewBdStore(dir, func(string, string, ...string) ([]byte, error) {
		t.Fatal("backing bd runner should not be called while opening doltlite read store")
		return nil, nil
	})
	store, err := NewDoltliteReadStore(dir, backing)
	if err != nil {
		t.Fatalf("NewDoltliteReadStore: %v", err)
	}
	defer store.CloseStore() //nolint:errcheck // test cleanup
}

func TestDoltliteReadStoreAttachesOperationalSQLiteDatabase(t *testing.T) {
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(filepath.Join(beadsDir, "doltlite"), 0o755); err != nil {
		t.Fatalf("mkdir doltlite dir: %v", err)
	}
	meta := []byte(`{
		"backend":"doltlite",
		"database":"doltlite",
		"dolt_database":"hq",
		"attached_databases":[{"alias":"ops","path":".gc/ops.sqlite"}]
	}`)
	if err := os.WriteFile(filepath.Join(beadsDir, "metadata.json"), meta, 0o600); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	db, err := sql.Open(doltliteSQLDriverName, filepath.Join(beadsDir, "doltlite", "hq.db")+"?_busy_timeout=10000")
	if err != nil {
		t.Fatalf("open doltlite fixture db: %v", err)
	}
	createTestDoltliteSchema(t, db)
	if err := db.Close(); err != nil {
		t.Fatalf("close fixture db: %v", err)
	}

	opsPath := filepath.Join(dir, ".gc", "ops.sqlite")
	if err := os.MkdirAll(filepath.Dir(opsPath), 0o755); err != nil {
		t.Fatalf("mkdir ops dir: %v", err)
	}
	ops, err := sql.Open(doltliteSQLDriverName, opsPath+"?_busy_timeout=10000")
	if err != nil {
		t.Fatalf("open ops sqlite fixture: %v", err)
	}
	if _, err := ops.Exec(`CREATE TABLE runtime_state (key TEXT PRIMARY KEY, value TEXT);
		INSERT INTO runtime_state (key, value) VALUES ('lease', 'unversioned')`); err != nil {
		t.Fatalf("seed ops sqlite fixture: %v", err)
	}
	if err := ops.Close(); err != nil {
		t.Fatalf("close ops fixture: %v", err)
	}

	backing := NewBdStore(dir, func(string, string, ...string) ([]byte, error) {
		t.Fatal("backing bd runner should not be called while opening doltlite read store")
		return nil, nil
	})
	store, err := NewDoltliteReadStore(dir, backing)
	if err != nil {
		t.Fatalf("NewDoltliteReadStore: %v", err)
	}
	defer store.CloseStore() //nolint:errcheck // test cleanup

	var value string
	if err := store.db.QueryRow(`SELECT value FROM ops.runtime_state WHERE key = 'lease'`).Scan(&value); err != nil {
		t.Fatalf("query attached ops database: %v", err)
	}
	if value != "unversioned" {
		t.Fatalf("attached ops value = %q, want unversioned", value)
	}
}

func TestDoltliteReadStoreSkipLabels(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	rows, err := store.List(ListQuery{
		Label:      "gc:session",
		SkipLabels: true,
	})
	if err != nil {
		t.Fatalf("List session beads: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("session rows = %d, want 1", len(rows))
	}
	if len(rows[0].Labels) != 0 {
		t.Fatalf("labels hydrated with SkipLabels=true: %v", rows[0].Labels)
	}
}

func TestDoltliteReadStoreHydratesParent(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	withParent, err := store.List(ListQuery{Type: "task", Sort: SortCreatedAsc})
	if err != nil {
		t.Fatalf("List tasks with parent: %v", err)
	}
	child := findTestBead(t, withParent, "gc-child")
	if child.ParentID != "gc-parent" {
		t.Fatalf("child parent = %q, want gc-parent", child.ParentID)
	}
}

func TestDoltliteReadStoreHandlesCanonicalDependencySchema(t *testing.T) {
	store, closeStore := newTestDoltliteReadStoreWithCanonicalDeps(t)
	defer closeStore()

	rows, err := store.List(ListQuery{Type: "task", Sort: SortCreatedAsc})
	if err != nil {
		t.Fatalf("List tasks with canonical deps schema: %v", err)
	}
	child := findTestBead(t, rows, "gc-child")
	if child.ParentID != "gc-parent" {
		t.Fatalf("child parent = %q, want gc-parent", child.ParentID)
	}

	down, err := store.DepList("gc-child", "down")
	if err != nil {
		t.Fatalf("DepList down with canonical deps schema: %v", err)
	}
	if len(down) != 1 || down[0].DependsOnID != "gc-parent" {
		t.Fatalf("down deps = %#v, want gc-parent", down)
	}

	up, err := store.DepList("gc-parent", "up")
	if err != nil {
		t.Fatalf("DepList up with canonical deps schema: %v", err)
	}
	if len(up) != 1 || up[0].IssueID != "gc-child" {
		t.Fatalf("up deps = %#v, want gc-child", up)
	}

	ready, err := store.Ready()
	if err != nil {
		t.Fatalf("Ready with canonical deps schema: %v", err)
	}
	if !hasTestBead(ready, "gc-ready") {
		t.Fatalf("Ready missing gc-ready: %#v", ready)
	}
	if hasTestBead(ready, "gc-blocked") {
		t.Fatalf("Ready included blocked issue: %#v", ready)
	}
}

func TestDoltliteReadStoreTypeFallbackCanSkipLabels(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	rows, err := store.List(ListQuery{
		Type:       "session",
		SkipLabels: true,
	})
	if err != nil {
		t.Fatalf("List type=session: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("type=session rows = %d, want 1", len(rows))
	}
	if rows[0].ID != "gc-session" {
		t.Fatalf("type=session row = %s, want gc-session", rows[0].ID)
	}
	if len(rows[0].Labels) != 0 {
		t.Fatalf("unexpected hydrated labels: %v", rows[0].Labels)
	}
}

func TestDoltliteReadStoreReadyUsesDoltlite(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	rows, err := store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if !hasTestBead(rows, "gc-ready") {
		t.Fatalf("Ready missing gc-ready: %#v", rows)
	}
	if hasTestBead(rows, "gc-session") {
		t.Fatalf("Ready included session bead: %#v", rows)
	}
	if hasTestBead(rows, "gc-blocked") {
		t.Fatalf("Ready included blocked bead: %#v", rows)
	}
}

func TestDoltliteReadStoreReadyCacheInvalidatesOnExternalWrite(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	initial, err := store.Ready()
	if err != nil {
		t.Fatalf("initial Ready: %v", err)
	}
	initialCount := len(initial)

	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup
	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-ready-external",
		Title:     "external ready",
		Status:    "open",
		IssueType: "task",
		CreatedAt: time.Now().UTC().Add(5 * time.Second),
	})

	after, err := store.Ready()
	if err != nil {
		t.Fatalf("Ready after external write: %v", err)
	}
	if len(after) != initialCount+1 {
		t.Fatalf("Ready() len = %d, want %d after external write", len(after), initialCount+1)
	}
	if !hasTestBead(after, "gc-ready-external") {
		t.Fatalf("Ready() missing external write bead: %#v", after)
	}
}

func TestDoltliteReadStoreReadyBlocksWorkflowDependencyTypes(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	for _, depType := range []string{"waits-for", "conditional-blocks"} {
		insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
			ID:        "gc-blocked-" + depType,
			Title:     "workflow blocked",
			Status:    "open",
			IssueType: "task",
			CreatedAt: time.Now().UTC().Add(time.Minute),
			Dependencies: []testDoltliteDependency{{
				DependsOnID: "gc-blocker",
				Type:        depType,
			}},
		})
	}

	rows, err := store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	for _, depType := range []string{"waits-for", "conditional-blocks"} {
		id := "gc-blocked-" + depType
		if hasTestBead(rows, id) {
			t.Fatalf("Ready included %s blocked by %s: %#v", id, depType, rows)
		}
	}
}

func TestDoltliteReadStoreReadyDefaultsMissingDependencyTypeToBlocks(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-empty-dep-type",
		Title:     "blocked by empty dependency type",
		Status:    "open",
		IssueType: "task",
		CreatedAt: time.Now().UTC().Add(time.Minute),
		Assignee:  "rig/missing-dep-type",
		Dependencies: []testDoltliteDependency{{
			DependsOnID: "gc-blocker",
			Type:        "",
		}},
	})
	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-null-dep-type",
		Title:     "blocked by null dependency type",
		Status:    "open",
		IssueType: "task",
		CreatedAt: time.Now().UTC().Add(2 * time.Minute),
		Assignee:  "rig/missing-dep-type",
	})
	if _, err := writer.Exec(`INSERT INTO dependencies (
		issue_id, depends_on_id, depends_on_issue_id, depends_on_wisp_id, depends_on_external, type
	) VALUES (?, ?, ?, '', '', NULL)`, "gc-null-dep-type", "gc-blocker", "gc-blocker"); err != nil {
		t.Fatalf("insert null dependency type: %v", err)
	}

	rows, err := store.Ready(ReadyQuery{Assignee: "rig/missing-dep-type"})
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("Ready included rows blocked by missing dependency types: %#v", rows)
	}
}

func TestDoltliteReadStoreReadyBlocksMissingTargets(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	for _, depType := range []string{"blocks", "waits-for", "conditional-blocks"} {
		insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
			ID:        "gc-missing-target-" + depType,
			Title:     "missing target blocked",
			Status:    "open",
			IssueType: "task",
			CreatedAt: time.Now().UTC().Add(time.Minute),
			Assignee:  "rig/missing-targets",
			Dependencies: []testDoltliteDependency{{
				DependsOnID: "gc-missing-" + depType,
				Type:        depType,
			}},
		})
	}

	rows, err := store.Ready(ReadyQuery{Assignee: "rig/missing-targets"})
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("Ready included beads with missing blockers: %#v", rows)
	}
}

func TestDoltliteReadStoreReadyFailedClosedDependencyStillBlocks(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-failed-blocker",
		Title:     "failed blocker",
		Status:    "closed",
		IssueType: "task",
		CreatedAt: time.Now().UTC().Add(time.Minute),
		Metadata: map[string]string{
			"gc.outcome": "fail",
		},
	})
	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-blocked-by-failed",
		Title:     "blocked by failed dependency",
		Status:    "open",
		IssueType: "task",
		CreatedAt: time.Now().UTC().Add(2 * time.Minute),
		Assignee:  "rig/failed-dependency",
		Dependencies: []testDoltliteDependency{{
			DependsOnID: "gc-failed-blocker",
			Type:        "blocks",
		}},
	})
	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-passed-blocker",
		Title:     "passed blocker",
		Status:    "closed",
		IssueType: "task",
		CreatedAt: time.Now().UTC().Add(3 * time.Minute),
		Metadata: map[string]string{
			"gc.outcome": "pass",
		},
	})
	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-released-by-passed",
		Title:     "released by passed dependency",
		Status:    "open",
		IssueType: "task",
		CreatedAt: time.Now().UTC().Add(4 * time.Minute),
		Assignee:  "rig/failed-dependency",
		Dependencies: []testDoltliteDependency{{
			DependsOnID: "gc-passed-blocker",
			Type:        "blocks",
		}},
	})

	rows, err := store.Ready(ReadyQuery{Assignee: "rig/failed-dependency"})
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if hasTestBead(rows, "gc-blocked-by-failed") {
		t.Fatalf("Ready included work blocked by failed closed dependency: %#v", rows)
	}
	if !hasTestBead(rows, "gc-released-by-passed") {
		t.Fatalf("Ready missing work released by passed closed dependency: %#v", rows)
	}
}

func TestDoltliteReadStoreReadyBlocksOpenWispTargets(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	for _, depType := range []string{"blocks", "waits-for", "conditional-blocks"} {
		insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
			ID:        "gc-wisp-blocked-" + depType,
			Title:     "wisp target blocked",
			Status:    "open",
			IssueType: "task",
			CreatedAt: time.Now().UTC().Add(time.Minute),
			Assignee:  "rig/wisp-blockers",
			Dependencies: []testDoltliteDependency{{
				DependsOnWispID: "gc-tier-wisp",
				Type:            depType,
			}},
		})
	}

	rows, err := store.Ready(ReadyQuery{Assignee: "rig/wisp-blockers"})
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("Ready included beads blocked by open wisps: %#v", rows)
	}
}

func TestDoltliteReadStoreReadyUsesTypedWispTargetWhenIDsCollide(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-collision-target",
		Title:     "closed issue sharing wisp id",
		Status:    "closed",
		IssueType: "task",
		CreatedAt: time.Now().UTC(),
	})
	insertTestDoltliteIssue(t, writer, "wisps", "wisp_labels", "wisp_dependencies", testDoltliteIssue{
		ID:        "gc-collision-target",
		Title:     "open wisp sharing issue id",
		Status:    "open",
		IssueType: "molecule",
		CreatedAt: time.Now().UTC(),
	})
	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-wisp-collision-blocked",
		Title:     "blocked by typed wisp target",
		Status:    "open",
		IssueType: "task",
		CreatedAt: time.Now().UTC().Add(time.Minute),
		Assignee:  "rig/wisp-collision",
		Dependencies: []testDoltliteDependency{{
			DependsOnWispID: "gc-collision-target",
			Type:            "blocks",
		}},
	})

	rows, err := store.Ready(ReadyQuery{Assignee: "rig/wisp-collision"})
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("Ready used closed issue status instead of open typed wisp target: %#v", rows)
	}
}

func TestDoltliteReadStoreReadyUsesIssueTargetWhenWispTargetEmpty(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-empty-wisp-closed-blocker",
		Title:     "closed issue blocker",
		Status:    "closed",
		IssueType: "task",
		CreatedAt: time.Now().UTC(),
	})
	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-empty-wisp-ready",
		Title:     "ready via closed issue target",
		Status:    "open",
		IssueType: "task",
		CreatedAt: time.Now().UTC().Add(time.Minute),
		Assignee:  "rig/empty-wisp-target",
		Dependencies: []testDoltliteDependency{{
			DependsOnID: "gc-empty-wisp-closed-blocker",
			Type:        "blocks",
		}},
	})

	rows, err := store.Ready(ReadyQuery{Assignee: "rig/empty-wisp-target"})
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if !hasTestBead(rows, "gc-empty-wisp-ready") {
		t.Fatalf("Ready missed issue whose empty wisp target should fall back to closed issue target: %#v", rows)
	}
}

func TestDoltliteReadStoreReadyHonorsLimit(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	rows, err := store.Ready(ReadyQuery{Limit: 1})
	if err != nil {
		t.Fatalf("Ready(limit=1): %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("Ready(limit=1) returned %d rows, want 1: %#v", len(rows), rows)
	}
}

func TestDoltliteReadStoreReadyLimitFindsReadyBehindBlockedWindow(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	depTypes := []string{"blocks", "waits-for", "conditional-blocks"}
	now := time.Now().UTC().Add(time.Minute)
	for i := 0; i < 100; i++ {
		insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
			ID:        fmt.Sprintf("gc-newer-blocked-%03d", i),
			Title:     "newer blocked",
			Status:    "open",
			IssueType: "task",
			CreatedAt: now.Add(time.Duration(i) * time.Second),
			Dependencies: []testDoltliteDependency{{
				DependsOnID: "gc-blocker",
				Type:        depTypes[i%len(depTypes)],
			}},
		})
	}

	rows, err := store.Ready(ReadyQuery{Limit: 1})
	if err != nil {
		t.Fatalf("Ready(limit=1): %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("Ready(limit=1) returned %d rows, want 1; rows=%#v", len(rows), rows)
	}
	if strings.HasPrefix(rows[0].ID, "gc-newer-blocked-") {
		t.Fatalf("Ready(limit=1) returned blocked row %#v", rows[0])
	}
}

func TestDoltliteReadStoreReadyOrdersPriorityBeforeCreated(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	now := time.Now().UTC()
	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-priority-low-newer",
		Title:     "low priority newer",
		Status:    "open",
		IssueType: "task",
		Priority:  2,
		CreatedAt: now.Add(time.Minute),
		Assignee:  "rig/priority",
	})
	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-priority-high-older",
		Title:     "high priority older",
		Status:    "open",
		IssueType: "task",
		Priority:  0,
		CreatedAt: now,
		Assignee:  "rig/priority",
	})

	rows, err := store.Ready(ReadyQuery{Assignee: "rig/priority", Limit: 1})
	if err != nil {
		t.Fatalf("Ready priority limit: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "gc-priority-high-older" {
		t.Fatalf("Ready priority order = %#v, want gc-priority-high-older first", rows)
	}
}

func TestDoltliteReadStoreHandlesNullDescription(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := writer.Exec(`
		INSERT INTO issues (
			id, title, status, issue_type, priority, created_at, updated_at,
			assignee, description, design, acceptance_criteria, notes, metadata
		)
		VALUES (?, ?, 'open', 'task', 2, ?, ?, ?, NULL, '', '', '', '{}')
	`, "gc-null-description", "null description", now, now, "rig/null-description"); err != nil {
		t.Fatalf("insert null description issue: %v", err)
	}

	got, err := store.Get("gc-null-description")
	if err != nil {
		t.Fatalf("Get null description: %v", err)
	}
	if got.Description != "" {
		t.Fatalf("Get description = %q, want empty string", got.Description)
	}

	listed, err := store.List(ListQuery{Assignee: "rig/null-description"})
	if err != nil {
		t.Fatalf("List null description: %v", err)
	}
	if len(listed) != 1 || listed[0].Description != "" {
		t.Fatalf("List null description rows = %#v, want one row with empty description", listed)
	}

	ready, err := store.Ready(ReadyQuery{Assignee: "rig/null-description"})
	if err != nil {
		t.Fatalf("Ready null description: %v", err)
	}
	if len(ready) != 1 || ready[0].Description != "" {
		t.Fatalf("Ready null description rows = %#v, want one row with empty description", ready)
	}
}

func TestDoltliteReadStoreHandlesMissingDependsOnExternalColumn(t *testing.T) {
	store, closeStore := newTestDoltliteReadStoreWithSchema(t, createTestDoltliteSchemaWithoutExternal)
	defer closeStore()

	got, err := store.Get("gc-session")
	if err != nil {
		t.Fatalf("Get session bead: %v", err)
	}
	if got.ID != "gc-session" || got.Type != "session" {
		t.Fatalf("Get session bead = %#v, want gc-session session", got)
	}

	ready, err := store.Ready(ReadyQuery{Assignee: "rig/ready-worker"})
	if err != nil {
		t.Fatalf("Ready with missing depends_on_external: %v", err)
	}
	if len(ready) == 0 {
		t.Fatalf("Ready with missing depends_on_external returned no rows, want seeded rows")
	}

	deps, err := store.DepList("gc-parent", "up")
	if err != nil {
		t.Fatalf("DepList with missing depends_on_external: %v", err)
	}
	if len(deps) != 1 || deps[0].IssueID != "gc-child" || deps[0].DependsOnID != "gc-parent" {
		t.Fatalf("DepList with missing depends_on_external = %#v, want gc-child -> gc-parent", deps)
	}
}

func TestDoltliteReadStoreHandlesSplitDependencySchema(t *testing.T) {
	store, closeStore := newTestDoltliteReadStoreWithSchema(t, createTestDoltliteSplitDependencySchema)
	defer closeStore()

	sessions, err := store.ListSessionBeads()
	if err != nil {
		t.Fatalf("ListSessionBeads with split dependency schema: %v", err)
	}
	if !hasTestBead(sessions, "gc-session") {
		t.Fatalf("ListSessionBeads with split dependency schema missing gc-session: %#v", sessions)
	}

	deps, err := store.DepList("gc-parent", "up")
	if err != nil {
		t.Fatalf("DepList with split dependency schema: %v", err)
	}
	if len(deps) != 1 || deps[0].IssueID != "gc-child" || deps[0].DependsOnID != "gc-parent" {
		t.Fatalf("DepList with split dependency schema = %#v, want gc-child -> gc-parent", deps)
	}

	newParent := "gc-blocker"
	if err := store.Update("gc-child", UpdateOpts{ParentID: &newParent}); err != nil {
		t.Fatalf("Update parent with split dependency schema: %v", err)
	}
	child, err := store.Get("gc-child")
	if err != nil {
		t.Fatalf("Get child after parent update: %v", err)
	}
	if child.ParentID != newParent {
		t.Fatalf("child parent = %q, want %q", child.ParentID, newParent)
	}

	if err := store.DepAdd("gc-tier-wisp", "gc-tier-issue", "blocks"); err != nil {
		t.Fatalf("DepAdd wisp with split dependency schema: %v", err)
	}
	wispDeps, err := store.DepList("gc-tier-wisp", "down")
	if err != nil {
		t.Fatalf("DepList wisp after DepAdd: %v", err)
	}
	if len(wispDeps) != 1 || wispDeps[0].DependsOnID != "gc-tier-issue" {
		t.Fatalf("wisp deps after DepAdd = %#v, want dependency on gc-tier-issue", wispDeps)
	}
	if err := store.DepRemove("gc-tier-wisp", "gc-tier-issue"); err != nil {
		t.Fatalf("DepRemove wisp with split dependency schema: %v", err)
	}
	wispDeps, err = store.DepList("gc-tier-wisp", "down")
	if err != nil {
		t.Fatalf("DepList wisp after DepRemove: %v", err)
	}
	if len(wispDeps) != 0 {
		t.Fatalf("wisp deps after DepRemove = %#v, want none", wispDeps)
	}
}

// TestDoltliteReadStoreBeforeFiltersRespectCutoff verifies that the CreatedBefore
// and UpdatedBefore list filters return only rows whose timestamps precede the
// cutoff. Timestamps are seeded in the store's canonical SQLite text format
// (doltliteSQLiteTime) because the before-filters compare with SQLite julianday()
// and parse with parseTimeString, both of which require ISO-8601 text. Binding a
// raw time.Time instead delegates formatting to the SQL driver. The explicit
// doltliteSQLiteTime formatting keeps this independent of driver details and
// preserves julianday() compatibility. See ga-p7ipsu.
func TestDoltliteReadStoreBeforeFiltersRespectCutoff(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	cutoff := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	for _, issue := range []struct {
		id        string
		createdAt time.Time
		updatedAt time.Time
	}{
		{id: "gc-native-time-before", createdAt: cutoff.Add(-time.Hour), updatedAt: cutoff.Add(-30 * time.Minute)},
		{id: "gc-native-time-after", createdAt: cutoff.Add(time.Hour), updatedAt: cutoff.Add(30 * time.Minute)},
	} {
		if _, err := writer.Exec(`INSERT INTO issues (
			id, title, status, issue_type, priority, created_at, updated_at,
			assignee, description, design, acceptance_criteria, notes, metadata
		) VALUES (?, ?, 'open', 'task', 2, ?, ?, 'rig/native-time', '', '', '', '', '{}')`,
			issue.id, issue.id, doltliteSQLiteTime(issue.createdAt), doltliteSQLiteTime(issue.updatedAt)); err != nil {
			t.Fatalf("insert native timestamp issue %s: %v", issue.id, err)
		}
	}

	createdRows, err := store.List(ListQuery{
		Assignee:      "rig/native-time",
		CreatedBefore: cutoff,
		Sort:          SortCreatedAsc,
		SkipLabels:    true,
	})
	if err != nil {
		t.Fatalf("List CreatedBefore: %v", err)
	}
	if got := testBeadIDs(createdRows); !slices.Equal(got, []string{"gc-native-time-before"}) {
		t.Fatalf("CreatedBefore ids = %v, want [gc-native-time-before]; rows=%#v", got, createdRows)
	}

	updatedRows, err := store.List(ListQuery{
		Assignee:      "rig/native-time",
		UpdatedBefore: cutoff,
		Sort:          SortCreatedAsc,
		SkipLabels:    true,
	})
	if err != nil {
		t.Fatalf("List UpdatedBefore: %v", err)
	}
	if got := testBeadIDs(updatedRows); !slices.Equal(got, []string{"gc-native-time-before"}) {
		t.Fatalf("UpdatedBefore ids = %v, want [gc-native-time-before]; rows=%#v", got, updatedRows)
	}
}

func TestDoltliteReadStoreCachesInvalidateOnWorkingSetWrites(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	sessions, err := store.ListSessionBeads()
	if err != nil {
		t.Fatalf("ListSessionBeads before write: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("session count before write = %d, want 1", len(sessions))
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := writer.Exec(`
		INSERT INTO issues (
			id, title, status, issue_type, priority, created_at, updated_at,
			description, design, acceptance_criteria, notes, metadata
		)
		VALUES (?, ?, 'open', 'session', 2, ?, ?, '', '', '', '', ?)
	`, "gc-session-2", "session 2", now, now, `{"session_name":"session-2"}`); err != nil {
		t.Fatalf("insert session through writer: %v", err)
	}

	sessions, err = store.ListSessionBeads()
	if err != nil {
		t.Fatalf("ListSessionBeads after write: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("session count after uncommitted write = %d, want 2", len(sessions))
	}

	ready, err := store.Ready()
	if err != nil {
		t.Fatalf("Ready before task write: %v", err)
	}
	if hasTestBead(ready, "gc-ready-2") {
		t.Fatalf("Ready unexpectedly found gc-ready-2 before insert: %#v", ready)
	}

	later := time.Now().UTC().Add(time.Second).Format(time.RFC3339Nano)
	if _, err := writer.Exec(`
		INSERT INTO issues (
			id, title, status, issue_type, priority, created_at, updated_at,
			description, design, acceptance_criteria, notes, metadata
		)
		VALUES (?, ?, 'open', 'task', 2, ?, ?, '', '', '', '', ?)
	`, "gc-ready-2", "ready 2", later, later, `{}`); err != nil {
		t.Fatalf("insert ready work through writer: %v", err)
	}

	ready, err = store.Ready()
	if err != nil {
		t.Fatalf("Ready after task write: %v", err)
	}
	if !hasTestBead(ready, "gc-ready-2") {
		t.Fatalf("Ready after task write missing gc-ready-2: %#v", ready)
	}
}

func TestDoltliteReadStoreCreatesNoHistoryWispsInProcess(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	store.BdStore = NewBdStore(t.TempDir(), func(string, string, ...string) ([]byte, error) {
		t.Fatal("DoltliteReadStore.Create(NoHistory) shelled out to bd")
		return nil, nil
	})

	created, err := store.Create(Bead{
		Title:     "order:rig/direct",
		Labels:    []string{"order-run:rig/direct", "gc:order-tracking"},
		NoHistory: true,
		Metadata:  map[string]string{"gc.order": "rig/direct"},
	})
	if err != nil {
		t.Fatalf("Create(NoHistory): %v", err)
	}
	if created.ID == "" || !created.NoHistory || created.Ephemeral {
		t.Fatalf("created bead = %#v, want no-history wisp with generated id", created)
	}

	rows, err := store.List(ListQuery{Label: "order-run:rig/direct", TierMode: TierWisps})
	if err != nil {
		t.Fatalf("List created wisp: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("created wisp rows = %d, want 1: %#v", len(rows), rows)
	}
	got := rows[0]
	if got.ID != created.ID || got.Metadata["gc.order"] != "rig/direct" || !got.NoHistory || got.Ephemeral {
		t.Fatalf("created wisp = %#v, want %#v metadata and no-history flag", got, created.ID)
	}
	if !slices.Contains(got.Labels, "gc:order-tracking") {
		t.Fatalf("created labels = %v, missing gc:order-tracking", got.Labels)
	}
}

func TestDoltliteReadStoreWispDependenciesStayInProcess(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	store.BdStore = NewBdStore(t.TempDir(), func(string, string, ...string) ([]byte, error) {
		t.Fatal("DoltliteReadStore wisp dependency mutation shelled out to bd")
		return nil, nil
	})

	if err := store.DepAdd("gc-tier-wisp", "gc-tier-issue", "relates-to"); err != nil {
		t.Fatalf("DepAdd wisp -> issue: %v", err)
	}
	deps, err := store.DepList("gc-tier-wisp", "down")
	if err != nil {
		t.Fatalf("DepList after DepAdd: %v", err)
	}
	if !hasTestDep(deps, "gc-tier-wisp", "gc-tier-issue", "relates-to") {
		t.Fatalf("deps after DepAdd = %#v, missing wisp -> issue relates-to", deps)
	}

	if err := store.DepAdd("gc-tier-wisp", "gc-tier-nohistory", "blocks"); err != nil {
		t.Fatalf("DepAdd wisp -> wisp: %v", err)
	}
	deps, err = store.DepList("gc-tier-wisp", "down")
	if err != nil {
		t.Fatalf("DepList after wisp target DepAdd: %v", err)
	}
	if !hasTestDep(deps, "gc-tier-wisp", "gc-tier-nohistory", "blocks") {
		t.Fatalf("deps after wisp target DepAdd = %#v, missing wisp -> wisp blocks", deps)
	}

	if err := store.DepRemove("gc-tier-wisp", "gc-tier-issue"); err != nil {
		t.Fatalf("DepRemove: %v", err)
	}
	deps, err = store.DepList("gc-tier-wisp", "down")
	if err != nil {
		t.Fatalf("DepList after DepRemove: %v", err)
	}
	if hasTestDep(deps, "gc-tier-wisp", "gc-tier-issue", "relates-to") {
		t.Fatalf("deps after DepRemove = %#v, still has removed dependency", deps)
	}
}

func TestDoltliteReadStoreDurableWritesStayInProcess(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	store.BdStore = NewBdStore(t.TempDir(), func(_, _ string, args ...string) ([]byte, error) {
		t.Fatalf("DoltliteReadStore durable mutation shelled out to bd %s", strings.Join(args, " "))
		return nil, nil
	})

	title := "tier issue updated"
	status := "in_progress"
	assignee := "rig/durable-worker"
	if err := store.Update("gc-tier-issue", UpdateOpts{
		Title:        &title,
		Status:       &status,
		Assignee:     &assignee,
		Labels:       []string{"durable-fastpath"},
		RemoveLabels: []string{"tier-test"},
		Metadata:     map[string]string{"state": "running"},
	}); err != nil {
		t.Fatalf("Update durable issue: %v", err)
	}
	got, err := store.Get("gc-tier-issue")
	if err != nil {
		t.Fatalf("Get updated durable issue: %v", err)
	}
	if got.Title != title || got.Status != status || got.Assignee != assignee || got.Metadata["state"] != "running" {
		t.Fatalf("updated durable issue = %#v", got)
	}
	if !slices.Contains(got.Labels, "durable-fastpath") || slices.Contains(got.Labels, "tier-test") {
		t.Fatalf("updated durable labels = %v", got.Labels)
	}

	if err := store.SetMetadataBatch("gc-tier-issue", map[string]string{
		"state": "done",
		"phase": "metadata",
	}); err != nil {
		t.Fatalf("SetMetadataBatch durable issue: %v", err)
	}
	got, err = store.Get("gc-tier-issue")
	if err != nil {
		t.Fatalf("Get metadata-updated durable issue: %v", err)
	}
	if got.Metadata["state"] != "done" || got.Metadata["phase"] != "metadata" {
		t.Fatalf("metadata-updated durable issue = %#v", got.Metadata)
	}

	if err := store.Close("gc-tier-issue"); err != nil {
		t.Fatalf("Close durable issue: %v", err)
	}
	got, err = store.Get("gc-tier-issue")
	if err != nil {
		t.Fatalf("Get closed durable issue: %v", err)
	}
	if got.Status != "closed" {
		t.Fatalf("closed durable issue status = %q", got.Status)
	}

	if err := store.Reopen("gc-tier-issue"); err != nil {
		t.Fatalf("Reopen durable issue: %v", err)
	}
	got, err = store.Get("gc-tier-issue")
	if err != nil {
		t.Fatalf("Get reopened durable issue: %v", err)
	}
	if got.Status != "open" {
		t.Fatalf("reopened durable issue status = %q", got.Status)
	}

	if err := store.DepAdd("gc-tier-wisp", "gc-tier-issue", "relates-to"); err != nil {
		t.Fatalf("seed cross-tier dependency: %v", err)
	}
	if err := store.Delete("gc-tier-issue"); err != nil {
		t.Fatalf("Delete durable issue: %v", err)
	}
	if _, err := store.Get("gc-tier-issue"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get deleted durable issue err = %v, want ErrNotFound", err)
	}
	deps, err := store.DepList("gc-tier-wisp", "down")
	if err != nil {
		t.Fatalf("DepList after durable Delete: %v", err)
	}
	if hasTestDep(deps, "gc-tier-wisp", "gc-tier-issue", "relates-to") {
		t.Fatalf("deps after durable Delete = %#v, still references deleted issue", deps)
	}
}

func TestDoltliteReadStoreDeletesWispsInProcess(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	store.BdStore = NewBdStore(t.TempDir(), func(string, string, ...string) ([]byte, error) {
		t.Fatal("DoltliteReadStore.Delete(wisp) shelled out to bd")
		return nil, nil
	})

	if err := store.DepAdd("gc-tier-nohistory", "gc-tier-wisp", "blocks"); err != nil {
		t.Fatalf("seed wisp dependency: %v", err)
	}
	if err := store.Delete("gc-tier-wisp"); err != nil {
		t.Fatalf("Delete wisp: %v", err)
	}
	rows, err := store.List(ListQuery{Label: "tier-test", TierMode: TierWisps, IncludeClosed: true})
	if err != nil {
		t.Fatalf("List after Delete: %v", err)
	}
	if hasTestBead(rows, "gc-tier-wisp") {
		t.Fatalf("wisp rows after Delete = %#v, still contains deleted wisp", rows)
	}
	deps, err := store.DepList("gc-tier-nohistory", "down")
	if err != nil {
		t.Fatalf("DepList after Delete: %v", err)
	}
	if hasTestDep(deps, "gc-tier-nohistory", "gc-tier-wisp", "blocks") {
		t.Fatalf("deps after Delete = %#v, still references deleted wisp", deps)
	}
}

func TestDoltliteReadStoreReadsOrderRunHotPaths(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	last, err := store.LastOrderRun("rig/sweep")
	if err != nil {
		t.Fatalf("LastOrderRun: %v", err)
	}
	if last.IsZero() {
		t.Fatal("LastOrderRun returned zero time")
	}

	open, err := store.HasOpenOrderRun("rig/sweep")
	if err != nil {
		t.Fatalf("HasOpenOrderRun(open): %v", err)
	}
	if open {
		t.Fatal("HasOpenOrderRun reported open for closed run")
	}

	open, err = store.HasOpenOrderRun("rig/active")
	if err != nil {
		t.Fatalf("HasOpenOrderRun(active): %v", err)
	}
	if !open {
		t.Fatal("HasOpenOrderRun did not find active run")
	}
}

func TestDoltliteReadStoreListsQueuedNudgeBeads(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	rows, err := store.List(ListQuery{
		Label: "gc:nudge",
	})
	if err != nil {
		t.Fatalf("List queued nudge beads: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("nudge rows = %d, want 1", len(rows))
	}
	got := rows[0]
	if got.ID != "gc-nudge" || got.Type != "chore" {
		t.Fatalf("nudge bead = %#v", got)
	}
	if got.Metadata["state"] != "queued" || got.Metadata["nudge_id"] != "nudge-1" {
		t.Fatalf("nudge metadata = %#v", got.Metadata)
	}
	if !slices.Contains(got.Labels, "agent:gastown/polecat") || !slices.Contains(got.Labels, "nudge:nudge-1") {
		t.Fatalf("nudge labels = %v", got.Labels)
	}
}

func TestDoltliteReadStoreFiltersNudgesByMetadata(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	rows, err := store.List(ListQuery{
		Type: "chore",
		Metadata: map[string]string{
			"target_session": "gastown__polecat-abc123",
			"state":          "queued",
		},
		SkipLabels: true,
	})
	if err != nil {
		t.Fatalf("List nudge by metadata: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "gc-nudge" {
		t.Fatalf("metadata rows = %#v, want gc-nudge", rows)
	}
	if len(rows[0].Labels) != 0 {
		t.Fatalf("labels hydrated with SkipLabels=true: %v", rows[0].Labels)
	}
}

func TestDoltliteReadStoreMetadataFilterFindsMatchBehindLimit(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	base := time.Now().UTC().Add(10 * time.Minute)
	for i := 0; i < 75; i++ {
		insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
			ID:        fmt.Sprintf("gc-metadata-skip-%02d", i),
			Title:     "newer non-match",
			Status:    "open",
			IssueType: "chore",
			CreatedAt: base.Add(time.Duration(i) * time.Second),
			Metadata: map[string]string{
				"state":          "queued",
				"target_session": "other-session",
			},
		})
	}
	insertTestDoltliteIssue(t, writer, "issues", "labels", "dependencies", testDoltliteIssue{
		ID:        "gc-metadata-match",
		Title:     "older match",
		Status:    "open",
		IssueType: "chore",
		CreatedAt: base.Add(-time.Hour),
		Metadata: map[string]string{
			"state":          "queued",
			"target_session": "metadata-sql-target",
		},
	})

	rows, err := store.List(ListQuery{
		Type: "chore",
		Metadata: map[string]string{
			"state":          "queued",
			"target_session": "metadata-sql-target",
		},
		Limit:      1,
		Sort:       SortCreatedDesc,
		SkipLabels: true,
	})
	if err != nil {
		t.Fatalf("List metadata with limit: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "gc-metadata-match" {
		t.Fatalf("metadata limited rows = %#v, want gc-metadata-match", rows)
	}
}

func TestDoltliteMetadataFilterPredicatesMatchStringValues(t *testing.T) {
	db, err := sql.Open(doltliteSQLDriverName, ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close() //nolint:errcheck // test cleanup
	if _, err := db.Exec(`CREATE TABLE rows (id TEXT, metadata TEXT)`); err != nil {
		t.Fatalf("create rows: %v", err)
	}
	for _, stmt := range []string{
		`INSERT INTO rows (id, metadata) VALUES ('match', '{"state":"queued","target_session":"worker-1"}')`,
		`INSERT INTO rows (id, metadata) VALUES ('spaced', '{"state": "queued", "target_session": "worker-1"}')`,
		`INSERT INTO rows (id, metadata) VALUES ('wrong', '{"state":"queued","target_session":"worker-2"}')`,
		`INSERT INTO rows (id, metadata) VALUES ('malformed', '{')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("insert fixture: %v", err)
		}
	}

	where, args := doltliteMetadataFilterPredicates(map[string]string{
		"state":          "queued",
		"target_session": "worker-1",
	})
	rows, err := db.Query(`SELECT id FROM rows i WHERE `+strings.Join(where, " AND ")+` ORDER BY id`, args...)
	if err != nil {
		t.Fatalf("query metadata predicates: %v", err)
	}
	defer rows.Close() //nolint:errcheck // test cleanup

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan id: %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}
	if !slices.Equal(ids, []string{"match", "spaced"}) {
		t.Fatalf("predicate ids = %v, want [match spaced]", ids)
	}
}

// TestDoltliteReadStoreTierModesIncludeWisps pins the storage-tier contract
// from query.go (TierMode) to the same shape TestBdStoreListStorageTierConformance
// pins for BdStore (#3045, #3444): TierIssues keeps history and no-history
// rows and drops only ephemeral ones; TierWisps keeps no-history and
// ephemeral rows; TierBoth unions everything.
func TestDoltliteReadStoreTierModesIncludeWisps(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	issues, err := store.List(ListQuery{Label: "tier-test", Sort: SortCreatedAsc})
	if err != nil {
		t.Fatalf("List issues tier: %v", err)
	}
	if got := testBeadIDs(issues); !slices.Equal(got, []string{"gc-tier-issue", "gc-tier-nohistory"}) {
		t.Fatalf("issues tier ids = %v, want [gc-tier-issue gc-tier-nohistory]; rows=%#v", got, issues)
	}
	noHistory := findTestBead(t, issues, "gc-tier-nohistory")
	if noHistory.Ephemeral || !noHistory.NoHistory {
		t.Fatalf("no-history row flags = %#v, want Ephemeral=false NoHistory=true", noHistory)
	}
	durable := findTestBead(t, issues, "gc-tier-issue")
	if durable.Ephemeral || durable.NoHistory {
		t.Fatalf("history row flags = %#v, want Ephemeral=false NoHistory=false", durable)
	}

	wisps, err := store.List(ListQuery{Label: "tier-test", TierMode: TierWisps, Sort: SortCreatedAsc})
	if err != nil {
		t.Fatalf("List wisps tier: %v", err)
	}
	if got := testBeadIDs(wisps); !slices.Equal(got, []string{"gc-tier-wisp", "gc-tier-nohistory"}) {
		t.Fatalf("wisps tier ids = %v, want [gc-tier-wisp gc-tier-nohistory]; rows=%#v", got, wisps)
	}
	if ephemeral := findTestBead(t, wisps, "gc-tier-wisp"); !ephemeral.Ephemeral || ephemeral.NoHistory {
		t.Fatalf("ephemeral row flags = %#v, want Ephemeral=true NoHistory=false", ephemeral)
	}

	both, err := store.List(ListQuery{Label: "tier-test", TierMode: TierBoth, Sort: SortCreatedAsc})
	if err != nil {
		t.Fatalf("List both tiers: %v", err)
	}
	if got := testBeadIDs(both); !slices.Equal(got, []string{"gc-tier-issue", "gc-tier-wisp", "gc-tier-nohistory"}) {
		t.Fatalf("both tier ids = %v, want [gc-tier-issue gc-tier-wisp gc-tier-nohistory]; rows=%#v", got, both)
	}
}

// TestDoltliteReadStoreLegacyWispsSchemaStaysEphemeralOnly pins backward
// compatibility for doltlite snapshots written before the wisps table carried
// the ephemeral/no_history storage-flag columns (beads migrations 0020/0023):
// without the discriminator every wisp row is ephemeral, so TierIssues must
// keep excluding the whole wisps table and TierWisps must surface its rows
// with Ephemeral=true.
func TestDoltliteReadStoreLegacyWispsSchemaStaysEphemeralOnly(t *testing.T) {
	store, closeStore := newLegacyTestDoltliteReadStore(t)
	defer closeStore()

	issues, err := store.List(ListQuery{Label: "tier-test", Sort: SortCreatedAsc})
	if err != nil {
		t.Fatalf("List issues tier: %v", err)
	}
	if got := testBeadIDs(issues); !slices.Equal(got, []string{"gc-legacy-issue"}) {
		t.Fatalf("issues tier ids = %v, want [gc-legacy-issue]; rows=%#v", got, issues)
	}

	wisps, err := store.List(ListQuery{Label: "tier-test", TierMode: TierWisps, Sort: SortCreatedAsc})
	if err != nil {
		t.Fatalf("List wisps tier: %v", err)
	}
	if len(wisps) != 1 || wisps[0].ID != "gc-legacy-wisp" || !wisps[0].Ephemeral {
		t.Fatalf("wisps tier rows = %#v, want only ephemeral gc-legacy-wisp", wisps)
	}

	both, err := store.List(ListQuery{Label: "tier-test", TierMode: TierBoth, Sort: SortCreatedAsc})
	if err != nil {
		t.Fatalf("List both tiers: %v", err)
	}
	if got := testBeadIDs(both); !slices.Equal(got, []string{"gc-legacy-issue", "gc-legacy-wisp"}) {
		t.Fatalf("both tier ids = %v, want [gc-legacy-issue gc-legacy-wisp]; rows=%#v", got, both)
	}
}

func TestDoltliteReadStoreGetFindsWisps(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	got, err := store.Get("gc-tier-wisp")
	if err != nil {
		t.Fatalf("Get wisp: %v", err)
	}
	if got.ID != "gc-tier-wisp" || !got.Ephemeral {
		t.Fatalf("Get wisp = %#v, want ephemeral gc-tier-wisp", got)
	}
}

func TestDoltliteReadStoreListSessionBeadsIncludesWisps(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	insertTestDoltliteIssue(t, writer, "wisps", "wisp_labels", "wisp_dependencies", testDoltliteIssue{
		ID:        "gc-wisp-session",
		Title:     "wisp session",
		Status:    "open",
		IssueType: "session",
		Labels:    []string{"gc:session"},
		Metadata:  map[string]string{"session_name": "wisp-session-1"},
	})

	rows, err := store.ListSessionBeads()
	if err != nil {
		t.Fatalf("ListSessionBeads: %v", err)
	}
	got := findTestBead(t, rows, "gc-wisp-session")
	if !got.Ephemeral || got.Type != "session" || got.Metadata["session_name"] != "wisp-session-1" {
		t.Fatalf("wisp session = %#v", got)
	}
}

func TestDoltliteReadStoreSetMetadataBatchUpdatesWisp(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	if err := store.SetMetadataBatch("gc-tier-wisp", map[string]string{"state": "start-pending"}); err != nil {
		t.Fatalf("SetMetadataBatch wisp: %v", err)
	}
	got, err := store.Get("gc-tier-wisp")
	if err != nil {
		t.Fatalf("Get wisp: %v", err)
	}
	if got.Metadata["kind"] != "wisp" || got.Metadata["state"] != "start-pending" {
		t.Fatalf("wisp metadata = %#v, want preserved kind and new state", got.Metadata)
	}
}

func TestDoltliteReadStoreSetMetadataBatchCoercesNonStringWispMetadata(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	if _, err := writer.Exec(
		`UPDATE wisps SET metadata = ? WHERE id = ?`,
		`{"kind":"wisp","gc.control_epoch":3,"configured_named_session":true}`,
		"gc-tier-wisp",
	); err != nil {
		t.Fatalf("seed raw wisp metadata: %v", err)
	}
	if err := store.SetMetadataBatch("gc-tier-wisp", map[string]string{"state": "running"}); err != nil {
		t.Fatalf("SetMetadataBatch wisp: %v", err)
	}
	got, err := store.Get("gc-tier-wisp")
	if err != nil {
		t.Fatalf("Get wisp: %v", err)
	}
	if got.Metadata["kind"] != "wisp" ||
		got.Metadata["gc.control_epoch"] != "3" ||
		got.Metadata["configured_named_session"] != "true" ||
		got.Metadata["state"] != "running" {
		t.Fatalf("wisp metadata = %#v, want coerced existing values and new state", got.Metadata)
	}
}

func TestDoltliteReadStoreConcurrentMetadataWritesPreserveBothKeys(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	dir := filepath.Dir(filepath.Dir(filepath.Dir(store.dbPath)))
	backing2 := NewBdStore(dir, func(string, string, ...string) ([]byte, error) {
		t.Fatal("backing bd runner should not be called by concurrent metadata regression test")
		return nil, nil
	})
	store2, err := NewDoltliteReadStore(dir, backing2)
	if err != nil {
		t.Fatalf("NewDoltliteReadStore second handle: %v", err)
	}
	defer func() { _ = store2.CloseStore() }()

	for i := 0; i < 12; i++ {
		stateKey := fmt.Sprintf("state-%d", i)
		phaseKey := fmt.Sprintf("phase-%d", i)
		start := make(chan struct{})
		errCh := make(chan error, 2)

		go func() {
			<-start
			errCh <- store.SetMetadataBatch("gc-tier-wisp", map[string]string{stateKey: "running"})
		}()
		go func() {
			<-start
			errCh <- store2.Update("gc-tier-wisp", UpdateOpts{Metadata: map[string]string{phaseKey: "dispatch"}})
		}()

		close(start)
		for j := 0; j < 2; j++ {
			if err := <-errCh; err != nil {
				t.Fatalf("concurrent metadata write round %d: %v", i, err)
			}
		}

		got, err := store.Get("gc-tier-wisp")
		if err != nil {
			t.Fatalf("Get after round %d: %v", i, err)
		}
		if got.Metadata["kind"] != "wisp" {
			t.Fatalf("round %d metadata lost kind: %#v", i, got.Metadata)
		}
		if got.Metadata[stateKey] != "running" || got.Metadata[phaseKey] != "dispatch" {
			t.Fatalf("round %d metadata lost concurrent keys %q/%q: %#v", i, stateKey, phaseKey, got.Metadata)
		}
	}
}

func TestDoltliteReadStoreCloseAndReopenWisp(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	if err := store.Close("gc-tier-wisp"); err != nil {
		t.Fatalf("Close wisp: %v", err)
	}
	closed, err := store.Get("gc-tier-wisp")
	if err != nil {
		t.Fatalf("Get closed wisp: %v", err)
	}
	if closed.Status != "closed" {
		t.Fatalf("closed wisp status = %q, want closed", closed.Status)
	}

	if err := store.Reopen("gc-tier-wisp"); err != nil {
		t.Fatalf("Reopen wisp: %v", err)
	}
	open, err := store.Get("gc-tier-wisp")
	if err != nil {
		t.Fatalf("Get reopened wisp: %v", err)
	}
	if open.Status != "open" {
		t.Fatalf("reopened wisp status = %q, want open", open.Status)
	}
}

func TestDoltliteReadStoreFiltersPluralAssigneesAcrossTiers(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	rows, err := store.List(ListQuery{
		Assignees: []string{"rig/ready-worker", "rig/wisp-worker"},
		TierMode:  TierBoth,
		Sort:      SortCreatedAsc,
	})
	if err != nil {
		t.Fatalf("List plural assignees: %v", err)
	}
	if got := testBeadIDs(rows); !slices.Equal(got, []string{"gc-assigned-ready", "gc-tier-wisp"}) {
		t.Fatalf("plural assignee ids = %v, want [gc-assigned-ready gc-tier-wisp]; rows=%#v", got, rows)
	}
	if !rows[1].Ephemeral {
		t.Fatalf("wisp row Ephemeral = false: %#v", rows[1])
	}
}

// TestDoltliteReadStoreLimitCutsDeterministicPrefixOnCreatedAtTies pins the
// (created_at, id) total order at the SQL layer (#3208): when rows share a
// created_at timestamp, a LIMIT-bounded read must cut the same prefix on
// every call. Without the id tiebreaker in ORDER BY, SQLite resolves ties in
// unspecified (rowid/insertion) order and the bounded subset is arbitrary.
func TestDoltliteReadStoreLimitCutsDeterministicPrefixOnCreatedAtTies(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	tie := doltliteSQLiteTime(time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC))
	// Insert in an order (c, a, b) that differs from both id directions so
	// an insertion-ordered tie-cut cannot accidentally match the contract.
	for _, id := range []string{"gc-tie-c", "gc-tie-a", "gc-tie-b"} {
		if _, err := writer.Exec(`INSERT INTO issues (
			id, title, status, issue_type, priority, created_at, updated_at,
			assignee, description, design, acceptance_criteria, notes, metadata
		) VALUES (?, ?, 'open', 'task', 2, ?, ?, 'rig/tie-order', '', '', '', '', '{}')`,
			id, id, tie, tie); err != nil {
			t.Fatalf("insert tie issue %s: %v", id, err)
		}
	}

	descTop2, err := store.List(ListQuery{
		Assignee:   "rig/tie-order",
		Sort:       SortCreatedDesc,
		Limit:      2,
		SkipLabels: true,
	})
	if err != nil {
		t.Fatalf("List desc limit 2: %v", err)
	}
	if got := testBeadIDs(descTop2); !slices.Equal(got, []string{"gc-tie-c", "gc-tie-b"}) {
		t.Fatalf("desc limit-2 ids = %v, want [gc-tie-c gc-tie-b]", got)
	}

	ascAll, err := store.List(ListQuery{
		Assignee:   "rig/tie-order",
		Sort:       SortCreatedAsc,
		SkipLabels: true,
	})
	if err != nil {
		t.Fatalf("List asc: %v", err)
	}
	if got := testBeadIDs(ascAll); !slices.Equal(got, []string{"gc-tie-a", "gc-tie-b", "gc-tie-c"}) {
		t.Fatalf("asc ids = %v, want [gc-tie-a gc-tie-b gc-tie-c]", got)
	}
}

// TestDoltliteReadStoreReadyLimitCutsDeterministicPrefixOnTies pins the same
// (#3208) tie-cut contract for the Ready path, whose custom ORDER BY
// (priority, created_at) also needs the id tiebreaker for a deterministic
// LIMIT prefix when rows share both keys.
func TestDoltliteReadStoreReadyLimitCutsDeterministicPrefixOnTies(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	tie := doltliteSQLiteTime(time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC))
	// Insert in an order (c, a, b) that differs from both id directions so
	// an insertion-ordered tie-cut cannot accidentally match the contract.
	for _, id := range []string{"gc-rtie-c", "gc-rtie-a", "gc-rtie-b"} {
		if _, err := writer.Exec(`INSERT INTO issues (
			id, title, status, issue_type, priority, created_at, updated_at,
			assignee, description, design, acceptance_criteria, notes, metadata
		) VALUES (?, ?, 'open', 'task', 2, ?, ?, 'rig/rtie-order', '', '', '', '', '{}')`,
			id, id, tie, tie); err != nil {
			t.Fatalf("insert ready tie issue %s: %v", id, err)
		}
	}

	top2, err := store.Ready(ReadyQuery{Assignee: "rig/rtie-order", Limit: 2})
	if err != nil {
		t.Fatalf("Ready limit 2: %v", err)
	}
	if got := testBeadIDs(top2); !slices.Equal(got, []string{"gc-rtie-a", "gc-rtie-b"}) {
		t.Fatalf("ready limit-2 ids = %v, want [gc-rtie-a gc-rtie-b]", got)
	}
}

func TestDoltliteReadStoreReadyFiltersPluralAssigneesAcrossTiers(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	rows, err := store.Ready(ReadyQuery{
		Assignees: []string{"rig/ready-worker", "rig/wisp-worker"},
		TierMode:  TierBoth,
	})
	if err != nil {
		t.Fatalf("Ready plural assignees: %v", err)
	}
	if got := testBeadIDs(rows); !slices.Equal(got, []string{"gc-assigned-ready", "gc-tier-wisp"}) {
		t.Fatalf("plural ready ids = %v, want [gc-assigned-ready gc-tier-wisp]; rows=%#v", got, rows)
	}
	if !rows[1].Ephemeral {
		t.Fatalf("wisp row Ephemeral = false: %#v", rows[1])
	}
}

func TestDoltliteReadStoreGCInternalReadWriteHarness(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()
	writer := openTestDoltliteWriter(t, store.db)
	defer writer.Close() //nolint:errcheck // test cleanup

	insertTestDoltliteIssue(t, writer, "wisps", "wisp_labels", "wisp_dependencies", testDoltliteIssue{
		ID:        "gc-session-wisp",
		Title:     "session wisp",
		Status:    "open",
		IssueType: "session",
		Assignee:  "rig/session-worker",
		Labels:    []string{"gc:session", "gc:runtime"},
		Metadata: map[string]string{
			"session_name": "session-wisp-1",
			"state":        "start-pending",
		},
		Dependencies: []testDoltliteDependency{{
			DependsOnID:      "gc-tier-issue",
			DependsOnIssueID: "gc-tier-issue",
			Type:             "relates-to",
		}, {
			DependsOnID:      "gc-parent",
			DependsOnIssueID: "gc-parent",
			Type:             "parent-child",
		}},
	})
	insertTestDoltliteIssue(t, writer, "wisps", "wisp_labels", "wisp_dependencies", testDoltliteIssue{
		ID:        "gc-ready-wisp",
		Title:     "ready wisp",
		Status:    "open",
		IssueType: "task",
		Assignee:  "rig/ready-wisp-worker",
	})

	assertIDs := func(name string, rows []Bead, want []string) {
		t.Helper()
		if got := testBeadIDs(rows); !slices.Equal(got, want) {
			t.Errorf("%s ids = %v, want %v; rows=%#v", name, got, want, rows)
		}
	}

	got, err := store.Get("gc-session-wisp")
	if err != nil {
		t.Fatalf("Get session wisp: %v", err)
	}
	if !got.Ephemeral || got.Type != "session" || got.Metadata["session_name"] != "session-wisp-1" {
		t.Fatalf("Get session wisp = %#v", got)
	}

	got, err = store.GetSessionBead("gc-session-wisp")
	if err != nil {
		t.Fatalf("GetSessionBead wisp id: %v", err)
	}
	if got.ID != "gc-session-wisp" || got.Metadata["state"] != "start-pending" {
		t.Fatalf("GetSessionBead by id = %#v", got)
	}

	got, err = store.GetSessionBead("session-wisp-1")
	if err != nil {
		t.Fatalf("GetSessionBead wisp session_name: %v", err)
	}
	if got.ID != "gc-session-wisp" {
		t.Fatalf("GetSessionBead by session_name = %#v", got)
	}

	sessions, err := store.ListSessionBeads()
	if err != nil {
		t.Fatalf("ListSessionBeads: %v", err)
	}
	if !hasTestBead(sessions, "gc-session") || !hasTestBead(sessions, "gc-session-wisp") {
		t.Fatalf("ListSessionBeads missing issue or wisp session: %#v", sessions)
	}

	labelRows, err := store.ListByLabel("gc:runtime", 10, WithBothTiers)
	if err != nil {
		t.Fatalf("ListByLabel both tiers: %v", err)
	}
	assertIDs("ListByLabel runtime", labelRows, []string{"gc-session-wisp"})

	metadataRows, err := store.ListByMetadata(map[string]string{"session_name": "session-wisp-1"}, 10, WithBothTiers)
	if err != nil {
		t.Fatalf("ListByMetadata both tiers: %v", err)
	}
	assertIDs("ListByMetadata session_name", metadataRows, []string{"gc-session-wisp"})

	assigneeRows, err := store.List(ListQuery{
		Assignee: "rig/session-worker",
		Status:   "open",
		Limit:    10,
		TierMode: TierBoth,
	})
	if err != nil {
		t.Fatalf("List by assignee wisp: %v", err)
	}
	assertIDs("List by assignee wisp", assigneeRows, []string{"gc-session-wisp"})

	bothRows, err := store.List(ListQuery{
		Assignees: []string{"rig/ready-worker", "rig/session-worker"},
		TierMode:  TierBoth,
		Sort:      SortCreatedAsc,
	})
	if err != nil {
		t.Fatalf("List both tiers by assignees: %v", err)
	}
	assertIDs("List both tiers by assignees", bothRows, []string{"gc-assigned-ready", "gc-session-wisp"})

	deps, err := store.DepList("gc-session-wisp", "down")
	if err != nil {
		t.Fatalf("DepList wisp down: %v", err)
	}
	depTypes := map[string]string{}
	for _, dep := range deps {
		if dep.IssueID == "gc-session-wisp" {
			depTypes[dep.DependsOnID] = dep.Type
		}
	}
	if depTypes["gc-tier-issue"] != "relates-to" || depTypes["gc-parent"] != "parent-child" {
		t.Fatalf("DepList wisp down = %#v", deps)
	}

	batchDeps, err := store.DepListBatch([]string{"gc-session-wisp", "gc-child"})
	if err != nil {
		t.Fatalf("DepListBatch mixed tiers: %v", err)
	}
	if len(batchDeps["gc-session-wisp"]) != 2 || len(batchDeps["gc-child"]) != 1 {
		t.Fatalf("DepListBatch mixed tiers = %#v", batchDeps)
	}

	children, err := store.Children("gc-parent", WithBothTiers)
	if err != nil {
		t.Fatalf("Children both tiers: %v", err)
	}
	assertIDs("Children both tiers", children, []string{"gc-child", "gc-session-wisp"})

	readyRows, err := store.Ready(ReadyQuery{Assignee: "rig/ready-wisp-worker", TierMode: TierBoth})
	if err != nil {
		t.Fatalf("Ready wisp assignee: %v", err)
	}
	assertIDs("Ready wisp assignee", readyRows, []string{"gc-ready-wisp"})

	if err := store.SetMetadataBatch("gc-session-wisp", map[string]string{
		"last_woke_at": "2026-06-07T16:00:00Z",
		"state":        "awake",
	}); err != nil {
		t.Fatalf("SetMetadataBatch wisp: %v", err)
	}
	got, err = store.Get("gc-session-wisp")
	if err != nil {
		t.Fatalf("Get metadata-updated wisp: %v", err)
	}
	if got.Metadata["state"] != "awake" || got.Metadata["last_woke_at"] == "" || got.Metadata["session_name"] != "session-wisp-1" {
		t.Fatalf("updated wisp metadata = %#v", got.Metadata)
	}

	if err := store.SetMetadata("gc-session-wisp", "last_seen_at", "2026-06-07T16:01:00Z"); err != nil {
		t.Errorf("SetMetadata wisp: %v", err)
	} else {
		got, err = store.Get("gc-session-wisp")
		if err != nil {
			t.Fatalf("Get single metadata-updated wisp: %v", err)
		}
		if got.Metadata["last_seen_at"] != "2026-06-07T16:01:00Z" || got.Metadata["state"] != "awake" {
			t.Errorf("single metadata-updated wisp metadata = %#v", got.Metadata)
		}
	}

	store.BdStore = NewBdStore(store.BdStore.dir, func(_, _ string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("unexpected backing bd runner: bd %s", strings.Join(args, " "))
	})
	updateStatus := "in_progress"
	updateAssignee := "rig/session-worker-2"
	if err := store.Update("gc-session-wisp", UpdateOpts{
		Status:   &updateStatus,
		Assignee: &updateAssignee,
		Metadata: map[string]string{"state": "running"},
	}); err != nil {
		t.Errorf("Update wisp session: %v", err)
	} else {
		got, err = store.Get("gc-session-wisp")
		if err != nil {
			t.Fatalf("Get Update-updated wisp: %v", err)
		}
		if got.Status != updateStatus || got.Assignee != updateAssignee || got.Metadata["state"] != "running" {
			t.Errorf("Update-updated wisp = %#v", got)
		}
	}

	closed, err := store.CloseAll([]string{"gc-session-wisp"}, map[string]string{
		"close_reason": "session create failed: aborted before creation_complete",
		"state":        "failed-create",
	})
	if err != nil {
		t.Fatalf("CloseAll wisp: %v", err)
	}
	if closed != 1 {
		t.Fatalf("CloseAll closed = %d, want 1", closed)
	}
	got, err = store.Get("gc-session-wisp")
	if err != nil {
		t.Fatalf("Get closed wisp: %v", err)
	}
	if got.Status != "closed" || got.Metadata["state"] != "failed-create" || got.Metadata["close_reason"] == "" {
		t.Fatalf("closed wisp = %#v", got)
	}

	if err := store.Reopen("gc-session-wisp"); err != nil {
		t.Fatalf("Reopen wisp: %v", err)
	}
	got, err = store.Get("gc-session-wisp")
	if err != nil {
		t.Fatalf("Get reopened wisp: %v", err)
	}
	if got.Status != "open" {
		t.Fatalf("reopened wisp status = %q, want open", got.Status)
	}
}

// This targets the DoltLite query helper directly so the regression can
// assert the generated SQL without depending on an end-to-end bd query.
func TestDoltliteReadStoreQueryIssuesOrderedUsesPerTableLimitAcrossTiers(t *testing.T) {
	recorder := newDoltliteQueryRecorder()
	driverName := "doltlite-limit-" + strings.NewReplacer("/", "-", " ", "-", ":", "-").Replace(t.Name())
	sql.Register(driverName, newDoltliteQueryDriver(recorder))
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("open fake doltlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	store := &DoltliteReadStore{db: db}
	rows, err := store.queryIssuesOrdered(ListQuery{
		SkipLabels: true,
		TierMode:   TierBoth,
	}, "", nil, 1, "")
	if err != nil {
		t.Fatalf("queryIssuesOrdered(TierBoth, limit=1): %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("queryIssuesOrdered(TierBoth, limit=1) returned %d rows, want 1", len(rows))
	}

	queries := recorder.queriesCopy()
	var issueQuery, wispQuery string
	for _, query := range queries {
		switch {
		case strings.Contains(query, "FROM issues i"):
			issueQuery = query
		case strings.Contains(query, "FROM wisps i"):
			wispQuery = query
		}
	}
	if issueQuery == "" || wispQuery == "" {
		t.Fatalf("recorded queries = %v, want issue and wisp ready scans", queries)
	}
	for _, query := range []string{issueQuery, wispQuery} {
		if !strings.Contains(query, "LIMIT 1") {
			t.Fatalf("query %q, want per-table LIMIT 1", query)
		}
	}
}

func TestDoltliteCachingStoreLiveFastReadDoesNotEraseDependencyCache(t *testing.T) {
	store, closeStore := newTestDoltliteReadStore(t)
	defer closeStore()

	cache := NewCachingStoreForTest(store, nil)
	if err := cache.Prime(context.Background()); err != nil {
		t.Fatalf("Prime: %v", err)
	}
	before, err := cache.DepList("gc-child", "down")
	if err != nil {
		t.Fatalf("DepList before fast read: %v", err)
	}
	if len(before) != 1 || before[0].DependsOnID != "gc-parent" {
		t.Fatalf("deps before fast read = %#v, want parent gc-parent", before)
	}

	if _, err := cache.List(ListQuery{
		Type:       "task",
		Live:       true,
		SkipLabels: true,
	}); err != nil {
		t.Fatalf("fast live List: %v", err)
	}

	after, err := cache.DepList("gc-child", "down")
	if err != nil {
		t.Fatalf("DepList after fast read: %v", err)
	}
	if len(after) != 1 || after[0].DependsOnID != "gc-parent" {
		t.Fatalf("deps after fast read = %#v, want parent gc-parent", after)
	}
}

func newTestDoltliteReadStore(t *testing.T) (*DoltliteReadStore, func()) {
	return newTestDoltliteReadStoreWithSchema(t, createTestDoltliteSchema)
}

func newTestDoltliteReadStoreWithSchema(t *testing.T, createSchema func(testing.TB, *sql.DB)) (*DoltliteReadStore, func()) {
	t.Helper()
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatalf("mkdir beads dir: %v", err)
	}
	meta := []byte(`{"backend":"doltlite","database":"doltlite","dolt_database":"hq"}`)
	if err := os.WriteFile(filepath.Join(beadsDir, "metadata.json"), meta, 0o600); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	dbDir := filepath.Join(beadsDir, "doltlite")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatalf("mkdir doltlite dir: %v", err)
	}
	dbPath := filepath.Join(dbDir, "hq.db")
	db, err := sql.Open(doltliteSQLDriverName, dbPath+"?_busy_timeout=10000")
	if err != nil {
		t.Fatalf("open doltlite fixture db: %v", err)
	}
	defer db.Close() //nolint:errcheck // test cleanup
	createSchema(t, db)

	now := time.Now().UTC()
	created := []testDoltliteIssue{
		{
			ID:          "gc-session",
			Title:       "session",
			Status:      "open",
			IssueType:   "session",
			CreatedAt:   now,
			Labels:      []string{"gc:session", "agent:test"},
			Metadata:    map[string]string{"session_name": "session-1"},
			Description: "session bead",
		},
		{
			ID:        "gc-parent",
			Title:     "parent",
			Status:    "open",
			IssueType: "task",
			CreatedAt: now,
		},
		{
			ID:        "gc-child",
			Title:     "child",
			Status:    "open",
			IssueType: "task",
			CreatedAt: now,
			Dependencies: []testDoltliteDependency{{
				DependsOnID: "gc-parent",
				Type:        "parent-child",
			}},
		},
		{
			ID:        "gc-ready",
			Title:     "ready",
			Status:    "open",
			IssueType: "task",
			CreatedAt: now,
		},
		{
			ID:        "gc-assigned-progress",
			Title:     "assigned progress",
			Status:    "in_progress",
			IssueType: "task",
			CreatedAt: now,
			Assignee:  "rig/worker",
		},
		{
			ID:        "gc-assigned-ready",
			Title:     "assigned ready",
			Status:    "open",
			IssueType: "task",
			CreatedAt: now,
			Assignee:  "rig/ready-worker",
		},
		{
			ID:        "gc-routed",
			Title:     "routed",
			Status:    "open",
			IssueType: "task",
			CreatedAt: now,
			Metadata:  map[string]string{"gc.routed_to": "rig/polecat"},
		},
		{
			ID:        "gc-blocker",
			Title:     "blocker",
			Status:    "open",
			IssueType: "task",
			CreatedAt: now,
		},
		{
			ID:        "gc-blocked",
			Title:     "blocked",
			Status:    "open",
			IssueType: "task",
			CreatedAt: now,
			Dependencies: []testDoltliteDependency{{
				DependsOnID: "gc-blocker",
				Type:        "blocks",
			}},
		},
		{
			ID:        "gc-nudge",
			Title:     "Queued nudge for gastown/polecat",
			Status:    "open",
			IssueType: "chore",
			CreatedAt: now,
			Labels:    []string{"gc:nudge", "agent:gastown/polecat", "nudge:nudge-1", "source:wait"},
			Metadata: map[string]string{
				"agent":          "gastown/polecat",
				"message":        "wait satisfied; continue",
				"nudge_id":       "nudge-1",
				"source":         "wait",
				"state":          "queued",
				"target_session": "gastown__polecat-abc123",
				"wait_bead_id":   "gc-wait",
			},
		},
		{
			ID:        "gc-wait",
			Title:     "Wait for dependency",
			Status:    "open",
			IssueType: "task",
			CreatedAt: now,
			Labels:    []string{"gc:wait"},
			Metadata: map[string]string{
				"nudge_id": "nudge-1",
				"state":    "ready",
			},
		},
		{
			ID:        "gc-order-closed",
			Title:     "order:rig/sweep",
			Status:    "closed",
			IssueType: "task",
			CreatedAt: now.Add(time.Second),
			Labels:    []string{"order-run:rig/sweep", "gc:order-tracking"},
		},
		{
			ID:        "gc-order-open",
			Title:     "order:rig/active",
			Status:    "open",
			IssueType: "task",
			CreatedAt: now.Add(2 * time.Second),
			Labels:    []string{"order-run:rig/active", "gc:order-tracking"},
		},
		{
			ID:        "gc-tier-issue",
			Title:     "tier issue",
			Status:    "open",
			IssueType: "task",
			CreatedAt: now.Add(3 * time.Second),
			Labels:    []string{"tier-test"},
		},
	}
	for _, issue := range created {
		insertTestDoltliteIssue(t, db, "issues", "labels", "dependencies", issue)
	}
	insertTestDoltliteIssue(t, db, "wisps", "wisp_labels", "wisp_dependencies", testDoltliteIssue{
		ID:        "gc-tier-wisp",
		Title:     "tier wisp",
		Status:    "open",
		IssueType: "task",
		CreatedAt: now.Add(4 * time.Second),
		Assignee:  "rig/wisp-worker",
		Labels:    []string{"tier-test"},
		Metadata:  map[string]string{"kind": "wisp"},
		Ephemeral: true,
	})
	insertTestDoltliteIssue(t, db, "wisps", "wisp_labels", "wisp_dependencies", testDoltliteIssue{
		ID:        "gc-tier-nohistory",
		Title:     "tier no-history",
		Status:    "open",
		IssueType: "task",
		CreatedAt: now.Add(5 * time.Second),
		Assignee:  "rig/nohistory-worker",
		Labels:    []string{"tier-test"},
		Metadata:  map[string]string{"kind": "no-history"},
		NoHistory: true,
	})

	backing := NewBdStore(dir, func(string, string, ...string) ([]byte, error) {
		t.Fatal("backing bd runner should not be called by doltlite read tests")
		return nil, nil
	})
	store, err := NewDoltliteReadStore(dir, backing)
	if err != nil {
		t.Fatalf("NewDoltliteReadStore: %v", err)
	}
	return store, func() { _ = store.CloseStore() }
}

func newTestDoltliteReadStoreWithCanonicalDeps(t *testing.T) (*DoltliteReadStore, func()) {
	t.Helper()
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(filepath.Join(beadsDir, "doltlite"), 0o755); err != nil {
		t.Fatalf("mkdir beads dir: %v", err)
	}
	meta := []byte(`{"backend":"doltlite","database":"doltlite","dolt_database":"hq"}`)
	if err := os.WriteFile(filepath.Join(beadsDir, "metadata.json"), meta, 0o600); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	dbPath := filepath.Join(beadsDir, "doltlite", "hq.db")
	db, err := sql.Open(doltliteSQLDriverName, dbPath+"?_busy_timeout=10000")
	if err != nil {
		t.Fatalf("open canonical fixture db: %v", err)
	}
	defer db.Close() //nolint:errcheck // test cleanup
	createTestDoltliteCanonicalDependencySchema(t, db)

	now := time.Now().UTC()
	for _, issue := range []testDoltliteIssue{
		{ID: "gc-parent", Title: "parent", Status: "open", IssueType: "task", CreatedAt: now},
		{ID: "gc-child", Title: "child", Status: "open", IssueType: "task", CreatedAt: now.Add(time.Second)},
		{ID: "gc-blocker", Title: "blocker", Status: "open", IssueType: "task", CreatedAt: now.Add(2 * time.Second)},
		{ID: "gc-blocked", Title: "blocked", Status: "open", IssueType: "task", CreatedAt: now.Add(3 * time.Second)},
		{ID: "gc-ready", Title: "ready", Status: "open", IssueType: "task", CreatedAt: now.Add(4 * time.Second)},
	} {
		insertTestDoltliteCanonicalIssue(t, db, issue)
	}
	insertTestDoltliteCanonicalDep(t, db, "gc-child", "gc-parent", "parent-child")
	insertTestDoltliteCanonicalDep(t, db, "gc-blocked", "gc-blocker", "blocks")

	backing := NewBdStore(dir, func(string, string, ...string) ([]byte, error) {
		t.Fatal("backing bd runner should not be called by canonical doltlite read tests")
		return nil, nil
	})
	store, err := NewDoltliteReadStore(dir, backing)
	if err != nil {
		t.Fatalf("NewDoltliteReadStore: %v", err)
	}
	return store, func() { _ = store.CloseStore() }
}

type testDoltliteDependency struct {
	DependsOnID       string
	DependsOnIssueID  string
	DependsOnWispID   string
	DependsOnExternal string
	Type              string
}

type testDoltliteIssue struct {
	ID           string
	Title        string
	Status       string
	IssueType    string
	Priority     int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Assignee     string
	Description  string
	Labels       []string
	Metadata     map[string]string
	Dependencies []testDoltliteDependency
	Ephemeral    bool
	NoHistory    bool
}

type doltliteQueryRecorder struct {
	mu      sync.Mutex
	queries []string
}

func newDoltliteQueryRecorder() *doltliteQueryRecorder {
	return &doltliteQueryRecorder{}
}

func (r *doltliteQueryRecorder) record(query string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queries = append(r.queries, query)
}

func (r *doltliteQueryRecorder) queriesCopy() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.queries...)
}

type doltliteQueryDriver struct {
	recorder *doltliteQueryRecorder
}

func newDoltliteQueryDriver(recorder *doltliteQueryRecorder) driver.Driver {
	return &doltliteQueryDriver{recorder: recorder}
}

func (d *doltliteQueryDriver) Open(string) (driver.Conn, error) {
	return &doltliteQueryConn{recorder: d.recorder}, nil
}

type doltliteQueryConn struct {
	recorder *doltliteQueryRecorder
}

func (c *doltliteQueryConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("prepare unsupported")
}

func (c *doltliteQueryConn) Close() error { return nil }

func (c *doltliteQueryConn) Begin() (driver.Tx, error) {
	return nil, fmt.Errorf("transactions unsupported")
}

func (c *doltliteQueryConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.recorder.record(query)
	switch {
	case strings.Contains(query, "PRAGMA table_info"):
		return &doltliteQueryRows{
			columns: []string{"cid", "name", "type", "notnull", "dflt_value", "pk"},
			rows: [][]driver.Value{
				{0, "issue_id", "TEXT", 0, nil, 0},
				{1, "depends_on_id", "TEXT", 0, nil, 0},
				{2, "depends_on_issue_id", "TEXT", 0, nil, 0},
				{3, "depends_on_wisp_id", "TEXT", 0, nil, 0},
				{4, "depends_on_external", "TEXT", 0, nil, 0},
				{5, "type", "TEXT", 0, nil, 0},
			},
		}, nil
	case strings.Contains(query, "sqlite_master"):
		return &doltliteQueryRows{
			columns: []string{"name"},
			rows:    [][]driver.Value{{"wisps"}},
		}, nil
	case strings.Contains(query, "FROM issues i"):
		return newDoltliteQueryRows("gc-issue", time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)), nil
	case strings.Contains(query, "FROM wisps i"):
		return newDoltliteQueryRows("gc-wisp", time.Date(2026, 6, 7, 11, 0, 0, 0, time.UTC)), nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
}

type doltliteQueryRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func newDoltliteQueryRows(id string, createdAt time.Time) driver.Rows {
	return &doltliteQueryRows{
		columns: []string{
			"id",
			"title",
			"status",
			"issue_type",
			"priority",
			"created_at",
			"updated_at",
			"assignee",
			"description",
			"metadata",
			"parent",
		},
		rows: [][]driver.Value{{
			id,
			id,
			"open",
			"task",
			nil,
			createdAt,
			createdAt,
			"",
			"",
			"{}",
			"",
		}},
	}
}

func (r *doltliteQueryRows) Columns() []string { return r.columns }

func (r *doltliteQueryRows) Close() error { return nil }

func (r *doltliteQueryRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

// createTestDoltliteSchema mirrors the snapshot schema the current DoltLite
// beads backend writes: the upstream wisps/no-history migrations (0020/0023)
// give both row tables ephemeral and no_history storage-flag columns.
// createLegacyTestDoltliteSchema covers snapshots from before those columns.
func createTestDoltliteSchema(t testing.TB, db *sql.DB) {
	createTestDoltliteSchemaWithExternal(t, db, true)
}

func createTestDoltliteSchemaWithoutExternal(t testing.TB, db *sql.DB) {
	createTestDoltliteSchemaWithExternal(t, db, false)
}

func createTestDoltliteSplitDependencySchema(t testing.TB, db *sql.DB) {
	t.Helper()
	const storageFlagColumns = `,
			ephemeral INTEGER DEFAULT 0,
			no_history INTEGER DEFAULT 0`
	createTestDoltliteSchemaWithRowAndDependencyColumns(t, db, storageFlagColumns, true, false)
}

func createTestDoltliteSchemaWithExternal(t testing.TB, db *sql.DB, includeExternal bool) {
	t.Helper()
	const storageFlagColumns = `,
			ephemeral INTEGER DEFAULT 0,
			no_history INTEGER DEFAULT 0`
	createTestDoltliteSchemaWithRowAndDependencyColumns(t, db, storageFlagColumns, includeExternal, true)
}

// createLegacyTestDoltliteSchema mirrors doltlite snapshots written before
// the wisps table carried storage-flag columns: every wisps row is ephemeral.
func createLegacyTestDoltliteSchema(t testing.TB, db *sql.DB) {
	t.Helper()
	createTestDoltliteSchemaWithRowAndDependencyColumns(t, db, "", true, true)
}

func createTestDoltliteSchemaWithRowColumns(t testing.TB, db *sql.DB, extraRowColumns string, includeExternal bool) {
	t.Helper()
	createTestDoltliteSchemaWithRowAndDependencyColumns(t, db, extraRowColumns, includeExternal, true)
}

func createTestDoltliteSchemaWithRowAndDependencyColumns(t testing.TB, db *sql.DB, extraRowColumns string, includeExternal, includeLegacyDependency bool) {
	t.Helper()
	depsColumns := []string{
		"issue_id TEXT",
		"depends_on_issue_id TEXT",
		"depends_on_wisp_id TEXT",
		"type TEXT",
	}
	if includeLegacyDependency {
		depsColumns = append(depsColumns[:1], append([]string{"depends_on_id TEXT"}, depsColumns[1:]...)...)
	}
	if includeExternal {
		insertAt := len(depsColumns) - 1
		depsColumns = append(depsColumns[:insertAt], append([]string{"depends_on_external TEXT"}, depsColumns[insertAt:]...)...)
	}
	depsSchema := "CREATE TABLE dependencies (\n\t\t\t" + strings.Join(depsColumns, ",\n\t\t\t") + "\n\t\t)"
	wispDepsSchema := "CREATE TABLE wisp_dependencies (\n\t\t\t" + strings.Join(depsColumns, ",\n\t\t\t") + "\n\t\t)"
	for _, stmt := range []string{
		`CREATE TABLE config (key TEXT PRIMARY KEY, value TEXT)`,
		`CREATE TABLE issues (
			id TEXT PRIMARY KEY,
			title TEXT,
			status TEXT,
			issue_type TEXT,
			priority INTEGER,
			created_at TEXT,
			updated_at TEXT,
			assignee TEXT,
			description TEXT,
			design TEXT,
			acceptance_criteria TEXT,
			notes TEXT,
			metadata TEXT` + extraRowColumns + `
		)`,
		`CREATE TABLE wisps (
			id TEXT PRIMARY KEY,
			title TEXT,
			status TEXT,
			issue_type TEXT,
			priority INTEGER,
			created_at TEXT,
			updated_at TEXT,
			assignee TEXT,
			description TEXT,
			design TEXT,
			acceptance_criteria TEXT,
			notes TEXT,
			metadata TEXT` + extraRowColumns + `
		)`,
		`CREATE TABLE labels (issue_id TEXT, label TEXT)`,
		`CREATE TABLE wisp_labels (issue_id TEXT, label TEXT)`,
		depsSchema,
		wispDepsSchema,
		`INSERT INTO config (key, value) VALUES ('issue_prefix', 'gc')`,
		`INSERT INTO config (key, value) VALUES ('types.custom', 'session,agent,role,rig,message,convoy,molecule,gate,merge-request')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("create test doltlite schema: %v\nstmt: %s", err, stmt)
		}
	}
}

func createTestDoltliteCanonicalDependencySchema(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, stmt := range []string{
		`CREATE TABLE config (key TEXT PRIMARY KEY, value TEXT)`,
		`CREATE TABLE issues (
			id TEXT PRIMARY KEY,
			title TEXT,
			status TEXT,
			issue_type TEXT,
			priority INTEGER,
			created_at TEXT,
			updated_at TEXT,
			assignee TEXT,
			description TEXT,
			metadata TEXT
		)`,
		`CREATE TABLE labels (issue_id TEXT, label TEXT)`,
		`CREATE TABLE dependencies (
			issue_id TEXT,
			depends_on_id TEXT,
			type TEXT
		)`,
		`INSERT INTO config (key, value) VALUES ('issue_prefix', 'gc')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("create canonical doltlite schema: %v\nstmt: %s", err, stmt)
		}
	}
}

func insertTestDoltliteCanonicalIssue(t *testing.T, db *sql.DB, issue testDoltliteIssue) {
	t.Helper()
	if issue.Status == "" {
		issue.Status = "open"
	}
	if issue.IssueType == "" {
		issue.IssueType = "task"
	}
	if issue.CreatedAt.IsZero() {
		issue.CreatedAt = time.Now().UTC()
	}
	if issue.UpdatedAt.IsZero() {
		issue.UpdatedAt = issue.CreatedAt
	}
	if _, err := db.Exec(`INSERT INTO issues (
		id, title, status, issue_type, priority, created_at, updated_at,
		assignee, description, metadata
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '{}')`,
		issue.ID,
		issue.Title,
		issue.Status,
		issue.IssueType,
		issue.Priority,
		issue.CreatedAt.Format(time.RFC3339Nano),
		issue.UpdatedAt.Format(time.RFC3339Nano),
		issue.Assignee,
		issue.Description,
	); err != nil {
		t.Fatalf("insert canonical issue %s: %v", issue.ID, err)
	}
}

func insertTestDoltliteCanonicalDep(t *testing.T, db *sql.DB, issueID, dependsOnID, depType string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO dependencies (issue_id, depends_on_id, type) VALUES (?, ?, ?)`, issueID, dependsOnID, depType); err != nil {
		t.Fatalf("insert canonical dep %s -> %s: %v", issueID, dependsOnID, err)
	}
}

func insertTestDoltliteIssue(t testing.TB, db *sql.DB, issueTable, labelTable, depTable string, issue testDoltliteIssue) {
	t.Helper()
	if issue.Status == "" {
		issue.Status = "open"
	}
	if issue.IssueType == "" {
		issue.IssueType = "task"
	}
	if issue.CreatedAt.IsZero() {
		issue.CreatedAt = time.Now().UTC()
	}
	if issue.UpdatedAt.IsZero() {
		issue.UpdatedAt = issue.CreatedAt
	}
	metadata := "{}"
	if len(issue.Metadata) > 0 {
		raw, err := json.Marshal(issue.Metadata)
		if err != nil {
			t.Fatalf("marshal metadata for %s: %v", issue.ID, err)
		}
		metadata = string(raw)
	}
	_, err := db.Exec(`INSERT INTO `+issueTable+` (
		id, title, status, issue_type, priority, created_at, updated_at,
		assignee, description, design, acceptance_criteria, notes, metadata,
		ephemeral, no_history
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '', '', '', ?, ?, ?)`,
		issue.ID,
		issue.Title,
		issue.Status,
		issue.IssueType,
		issue.Priority,
		issue.CreatedAt.Format(time.RFC3339Nano),
		issue.UpdatedAt.Format(time.RFC3339Nano),
		issue.Assignee,
		issue.Description,
		metadata,
		boolToTestInt(issue.Ephemeral),
		boolToTestInt(issue.NoHistory),
	)
	if err != nil {
		t.Fatalf("insert %s into %s: %v", issue.ID, issueTable, err)
	}
	for _, label := range issue.Labels {
		if _, err := db.Exec(`INSERT INTO `+labelTable+` (issue_id, label) VALUES (?, ?)`, issue.ID, label); err != nil {
			t.Fatalf("insert label %s for %s: %v", label, issue.ID, err)
		}
	}
	for _, dep := range issue.Dependencies {
		dependsOnIssueID := dep.DependsOnIssueID
		if dependsOnIssueID == "" && dep.DependsOnWispID == "" && dep.DependsOnExternal == "" {
			dependsOnIssueID = dep.DependsOnID
		}
		columns := []string{"issue_id"}
		values := []any{issue.ID}
		if testDoltliteTableHasColumn(t, db, depTable, "depends_on_id") {
			columns = append(columns, "depends_on_id")
			values = append(values, dep.DependsOnID)
		}
		if testDoltliteTableHasColumn(t, db, depTable, "depends_on_issue_id") {
			columns = append(columns, "depends_on_issue_id")
			values = append(values, dependsOnIssueID)
		}
		if testDoltliteTableHasColumn(t, db, depTable, "depends_on_wisp_id") {
			columns = append(columns, "depends_on_wisp_id")
			values = append(values, dep.DependsOnWispID)
		}
		if testDoltliteTableHasColumn(t, db, depTable, "depends_on_external") {
			columns = append(columns, "depends_on_external")
			values = append(values, dep.DependsOnExternal)
		}
		columns = append(columns, "type")
		values = append(values, dep.Type)
		placeholders := strings.TrimRight(strings.Repeat("?,", len(columns)), ",")
		if _, err := db.Exec(`INSERT INTO `+depTable+` (`+strings.Join(columns, ", ")+`) VALUES (`+placeholders+`)`, values...); err != nil {
			t.Fatalf("insert dep %s -> %s: %v", issue.ID, dep.DependsOnID, err)
		}
	}
}

func boolToTestInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// newLegacyTestDoltliteReadStore builds a read store over a pre-storage-flag
// snapshot (no ephemeral/no_history columns) seeded with one durable issue
// and one wisps row, both labeled tier-test.
func newLegacyTestDoltliteReadStore(t *testing.T) (*DoltliteReadStore, func()) {
	t.Helper()
	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatalf("mkdir beads dir: %v", err)
	}
	meta := []byte(`{"backend":"doltlite","database":"doltlite","dolt_database":"hq"}`)
	if err := os.WriteFile(filepath.Join(beadsDir, "metadata.json"), meta, 0o600); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	dbDir := filepath.Join(beadsDir, "doltlite")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatalf("mkdir doltlite dir: %v", err)
	}
	dbPath := filepath.Join(dbDir, "hq.db")
	db, err := sql.Open(doltliteSQLDriverName, dbPath+"?_busy_timeout=10000")
	if err != nil {
		t.Fatalf("open legacy doltlite fixture db: %v", err)
	}
	defer db.Close() //nolint:errcheck // test cleanup
	createLegacyTestDoltliteSchema(t, db)

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{
			`INSERT INTO issues (
			id, title, status, issue_type, priority, created_at, updated_at,
			assignee, description, design, acceptance_criteria, notes, metadata
		) VALUES (?, ?, 'open', 'task', 2, ?, ?, '', '', '', '', '', '{}')`,
			[]any{"gc-legacy-issue", "legacy issue", now, now},
		},
		{`INSERT INTO labels (issue_id, label) VALUES ('gc-legacy-issue', 'tier-test')`, nil},
		{
			`INSERT INTO wisps (
			id, title, status, issue_type, priority, created_at, updated_at,
			assignee, description, design, acceptance_criteria, notes, metadata
		) VALUES (?, ?, 'open', 'task', 2, ?, ?, '', '', '', '', '', '{}')`,
			[]any{"gc-legacy-wisp", "legacy wisp", now, now},
		},
		{`INSERT INTO wisp_labels (issue_id, label) VALUES ('gc-legacy-wisp', 'tier-test')`, nil},
	} {
		if _, err := db.Exec(stmt.sql, stmt.args...); err != nil {
			t.Fatalf("seed legacy doltlite fixture: %v\nstmt: %s", err, stmt.sql)
		}
	}

	backing := NewBdStore(dir, func(string, string, ...string) ([]byte, error) {
		t.Fatal("backing bd runner should not be called by doltlite read tests")
		return nil, nil
	})
	store, err := NewDoltliteReadStore(dir, backing)
	if err != nil {
		t.Fatalf("NewDoltliteReadStore: %v", err)
	}
	return store, func() { _ = store.CloseStore() }
}

func testDoltliteTableHasColumn(t testing.TB, db *sql.DB, table, column string) bool {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + quoteSQLiteIdentifier(table) + `)`)
	if err != nil {
		t.Fatalf("inspect %s columns: %v", table, err)
	}
	defer rows.Close() //nolint:errcheck // best-effort cleanup
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
			t.Fatalf("scan %s columns: %v", table, err)
		}
		if columnName == column {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate %s columns: %v", table, err)
	}
	return false
}

func testBeadIDs(rows []Bead) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func findTestBead(t *testing.T, rows []Bead, id string) Bead {
	t.Helper()
	for _, row := range rows {
		if row.ID == id {
			return row
		}
	}
	t.Fatalf("missing bead %s in %#v", id, rows)
	return Bead{}
}

func hasTestBead(rows []Bead, id string) bool {
	for _, row := range rows {
		if row.ID == id {
			return true
		}
	}
	return false
}

func hasTestDep(rows []Dep, issueID, dependsOnID, depType string) bool {
	for _, row := range rows {
		if row.IssueID == issueID && row.DependsOnID == dependsOnID && row.Type == depType {
			return true
		}
	}
	return false
}

func openTestDoltliteWriter(t *testing.T, readDB *sql.DB) *sql.DB {
	t.Helper()
	rows, err := readDB.Query("PRAGMA database_list")
	if err != nil {
		t.Fatalf("query database list: %v", err)
	}
	defer rows.Close() //nolint:errcheck // test cleanup

	var dbPath string
	for rows.Next() {
		var seq int
		var name, file string
		if err := rows.Scan(&seq, &name, &file); err != nil {
			t.Fatalf("scan database list: %v", err)
		}
		if name == "main" {
			dbPath = file
			break
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read database list: %v", err)
	}
	if dbPath == "" {
		t.Fatal("main database path not found")
	}

	writer, err := sql.Open(doltliteSQLDriverName, "file:"+dbPath+"?mode=rw&_busy_timeout=10000")
	if err != nil {
		t.Fatalf("open writable doltlite db: %v", err)
	}
	return writer
}
