package beads_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gastownhall/gascity/internal/beads"
	_ "modernc.org/sqlite" // pure-Go SQLite driver, safe in non-CGO builds
)

func TestBdStoreCloseUsesLocalDoltliteWrite(t *testing.T) {
	dir, db := newDoltliteBdStoreFixture(t, "bd-1", "open")
	store := beads.NewBdStore(dir, func(_, name string, args ...string) ([]byte, error) {
		if name != "bd" || len(args) < 3 || args[0] != "show" || args[1] != "--json" || args[2] != "bd-1" {
			t.Fatalf("unexpected runner call: %s %v", name, args)
		}
		return []byte(`[{"id":"bd-1","title":"t","status":"open","issue_type":"task","created_at":"2026-06-30T10:00:00Z"}]`), nil
	})

	if err := store.Close("bd-1"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := readDoltliteStatus(t, db, "issues", "bd-1"); got != "closed" {
		t.Fatalf("status after Close = %q, want closed", got)
	}
}

func TestBdStoreUpdateStatusClosedUsesLocalDoltliteWrite(t *testing.T) {
	dir, db := newDoltliteBdStoreFixture(t, "bd-2", "open")
	store := beads.NewBdStore(dir, func(_, name string, args ...string) ([]byte, error) {
		t.Fatalf("unexpected runner call: %s %v", name, args)
		return nil, nil
	})

	status := "closed"
	if err := store.Update("bd-2", beads.UpdateOpts{Status: &status}); err != nil {
		t.Fatalf("Update(status=closed): %v", err)
	}
	if got := readDoltliteStatus(t, db, "issues", "bd-2"); got != "closed" {
		t.Fatalf("status after Update = %q, want closed", got)
	}
}

func TestBdStoreReopenUsesLocalDoltliteWrite(t *testing.T) {
	dir, db := newDoltliteBdStoreFixture(t, "bd-3", "closed")
	store := beads.NewBdStore(dir, func(_, name string, args ...string) ([]byte, error) {
		t.Fatalf("unexpected runner call: %s %v", name, args)
		return nil, nil
	})

	if err := store.Reopen("bd-3"); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if got := readDoltliteStatus(t, db, "issues", "bd-3"); got != "open" {
		t.Fatalf("status after Reopen = %q, want open", got)
	}
}

func newDoltliteBdStoreFixture(t *testing.T, id, status string) (string, *sql.DB) {
	t.Helper()

	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads")
	dbDir := filepath.Join(beadsDir, "doltlite")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatalf("mkdir doltlite dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(beadsDir, "metadata.json"), []byte(`{"backend":"doltlite","database":"doltlite","dolt_database":"hq"}`), 0o600); err != nil {
		t.Fatalf("write metadata.json: %v", err)
	}

	dbPath := filepath.Join(dbDir, "hq.db")
	db, err := sql.Open("sqlite", dbPath+"?_busy_timeout=10000")
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, stmt := range []string{
		`CREATE TABLE issues (id TEXT PRIMARY KEY, status TEXT, updated_at TEXT)`,
		`CREATE TABLE wisps (id TEXT PRIMARY KEY, status TEXT, updated_at TEXT)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO issues (id, status, updated_at) VALUES (?, ?, ?)`, id, status, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatalf("insert issue: %v", err)
	}
	return dir, db
}

func readDoltliteStatus(t *testing.T, db *sql.DB, table, id string) string {
	t.Helper()
	var status string
	if err := db.QueryRow(`SELECT status FROM `+table+` WHERE id = ?`, id).Scan(&status); err != nil {
		t.Fatalf("query status: %v", err)
	}
	return status
}
