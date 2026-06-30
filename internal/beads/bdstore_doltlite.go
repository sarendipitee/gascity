package beads

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, safe in non-CGO builds
)

func (s *BdStore) tryDoltliteStatusOnlyUpdate(id string, opts UpdateOpts) (bool, error) {
	if !s.doltliteLocalWritesAvailable() || opts.Status == nil || !isStatusOnlyUpdate(opts) {
		return false, nil
	}
	if err := s.doltliteSetStatus(id, strings.TrimSpace(*opts.Status)); err != nil {
		return true, err
	}
	return true, nil
}

func (s *BdStore) doltliteClose(id, _ string) (bool, error) {
	if !s.doltliteLocalWritesAvailable() {
		return false, nil
	}
	if err := s.doltliteSetStatus(id, "closed"); err != nil {
		return true, err
	}
	return true, nil
}

func (s *BdStore) doltliteReopen(id string) (bool, error) {
	if !s.doltliteLocalWritesAvailable() {
		return false, nil
	}
	if err := s.doltliteSetStatus(id, "open"); err != nil {
		return true, err
	}
	return true, nil
}

func (s *BdStore) doltliteCloseAll(ids []string, metadata map[string]string) (int, bool, error) {
	if !s.doltliteLocalWritesAvailable() {
		return 0, false, nil
	}
	if len(ids) == 0 {
		return 0, true, nil
	}
	if len(metadata) > 0 {
		if err := s.setMetadataBatchAll(ids, metadata); err != nil {
			return 0, true, err
		}
	}
	reason := strings.TrimSpace(metadata["close_reason"])
	closed := 0
	for _, id := range ids {
		if _, err := s.doltliteClose(id, reason); err != nil {
			return closed, true, err
		}
		closed++
	}
	return closed, true, nil
}

func isStatusOnlyUpdate(opts UpdateOpts) bool {
	return opts.Title == nil &&
		opts.Type == nil &&
		opts.Priority == nil &&
		opts.Description == nil &&
		opts.ParentID == nil &&
		opts.Assignee == nil &&
		len(opts.Labels) == 0 &&
		len(opts.RemoveLabels) == 0 &&
		len(opts.Metadata) == 0
}

func (s *BdStore) doltliteLocalWritesAvailable() bool {
	if !s.isDoltliteBackend() {
		return false
	}
	dbPath, err := s.doltliteDBPath()
	if err != nil {
		return false
	}
	info, err := os.Stat(dbPath)
	return err == nil && !info.IsDir()
}

func (s *BdStore) doltliteSetStatus(id, status string) error {
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("updating bead %q: empty status", id)
	}
	db, err := s.openDoltliteWriter()
	if err != nil {
		return fmt.Errorf("updating bead %q: open doltlite writer: %w", id, err)
	}
	defer db.Close() //nolint:errcheck // best-effort cleanup

	ctx, cancel := context.WithTimeout(context.Background(), bdCommandTimeout)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("updating bead %q: begin doltlite transaction: %w", id, err)
	}
	defer func() { _ = tx.Rollback() }()

	table, currentStatus, err := doltliteStatusRowForID(ctx, tx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("updating bead %q: %w", id, ErrNotFound)
		}
		return fmt.Errorf("updating bead %q: resolve doltlite row: %w", id, err)
	}
	if currentStatus == status {
		return nil
	}

	if _, err := tx.ExecContext(ctx,
		fmt.Sprintf("UPDATE %s SET status = ?, updated_at = ? WHERE id = ?", table),
		status, time.Now().UTC().Format(time.RFC3339Nano), id); err != nil {
		return fmt.Errorf("updating bead %q: doltlite status update: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("updating bead %q: commit doltlite status update: %w", id, err)
	}
	return nil
}

func (s *BdStore) openDoltliteWriter() (*sql.DB, error) {
	dbPath, err := s.doltliteDBPath()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=rw&_busy_timeout=10000")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (s *BdStore) doltliteDBPath() (string, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, ".beads", "metadata.json"))
	if err != nil {
		return "", err
	}
	var meta struct {
		Backend      string `json:"backend"`
		Database     string `json:"database"`
		DoltDatabase string `json:"dolt_database"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", err
	}
	if !isDoltliteMetadata(meta.Backend, meta.Database) {
		return "", fmt.Errorf("not a doltlite beads store")
	}
	dbName := strings.TrimSpace(meta.DoltDatabase)
	if dbName == "" || strings.EqualFold(dbName, "doltlite") {
		dbName = strings.TrimSpace(meta.Database)
	}
	if dbName == "" || strings.EqualFold(dbName, "doltlite") {
		dbName = "hq"
	}
	return filepath.Join(s.dir, ".beads", "doltlite", dbName+".db"), nil
}

func doltliteStatusRowForID(ctx context.Context, tx *sql.Tx, id string) (string, string, error) {
	var foundTable, foundStatus string
	for _, table := range []string{"issues", "wisps"} {
		var status string
		err := tx.QueryRowContext(ctx, fmt.Sprintf("SELECT status FROM %s WHERE id = ?", table), id).Scan(&status)
		switch err {
		case nil:
			if foundTable != "" {
				return "", "", fmt.Errorf("duplicate bead id %q across doltlite tiers", id)
			}
			foundTable = table
			foundStatus = status
		case sql.ErrNoRows:
			continue
		default:
			return "", "", err
		}
	}
	if foundTable == "" {
		return "", "", ErrNotFound
	}
	return foundTable, foundStatus, nil
}
