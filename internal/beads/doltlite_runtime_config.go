package beads

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RepairDoltliteRuntimeConfigFile writes runtime config keys directly into a
// DoltLite database file through the linked sqlite driver.
func RepairDoltliteRuntimeConfigFile(parent context.Context, dbPath string, values map[string]string) error {
	if strings.TrimSpace(dbPath) == "" {
		return fmt.Errorf("doltlite runtime config: empty db path")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("doltlite runtime config: create db dir: %w", err)
	}

	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	db, err := sql.Open(doltliteSQLDriverName, "file:"+dbPath+"?mode=rwc&_busy_timeout=10000")
	if err != nil {
		return fmt.Errorf("doltlite runtime config: open db: %w", err)
	}
	defer db.Close() //nolint:errcheck // best-effort cleanup
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("doltlite runtime config: ping db: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS config ("key" TEXT PRIMARY KEY, value TEXT)`); err != nil {
		return fmt.Errorf("doltlite runtime config: ensure config table: %w", err)
	}

	for _, key := range []string{"issue_prefix", "types.custom"} {
		value := strings.TrimSpace(values[key])
		if value == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, `REPLACE INTO config ("key", value) VALUES (?, ?)`, key, value); err != nil {
			return fmt.Errorf("doltlite runtime config: set %s: %w", key, err)
		}
		var got string
		if err := db.QueryRowContext(ctx, `SELECT value FROM config WHERE "key" = ?`, key).Scan(&got); err != nil {
			return fmt.Errorf("doltlite runtime config: verify %s: %w", key, err)
		}
		if strings.TrimSpace(got) != value {
			return fmt.Errorf("doltlite runtime config: verify %s got %q want %q", key, strings.TrimSpace(got), value)
		}
	}
	return nil
}
