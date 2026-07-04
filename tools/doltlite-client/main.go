//go:build gascity_doltlite_lib

// Package main provides a small diagnostic client for DoltLite-backed beads stores.
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type metadataFile struct {
	Backend      string `json:"backend"`
	Database     string `json:"database"`
	DoltDatabase string `json:"dolt_database"`
}

type client struct {
	db   *sql.DB
	path string
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "doltlite-client: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("doltlite-client", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	city := fs.String("city", ".", "city workspace root")
	dbPath := fs.String("db", "", "direct DoltLite database path; overrides -city metadata")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		usage(fs)
		return nil
	}

	c, err := openClient(*city, *dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = c.db.Close() }()

	switch rest[0] {
	case "info":
		return c.info(ctx)
	case "query":
		if len(rest) < 2 {
			return errors.New("query requires SQL")
		}
		return c.query(ctx, rest[1], rest[2:]...)
	case "exec":
		if len(rest) < 2 {
			return errors.New("exec requires SQL")
		}
		return c.exec(ctx, rest[1], rest[2:]...)
	case "show":
		if len(rest) != 2 {
			return errors.New("show requires bead id")
		}
		return c.show(ctx, rest[1])
	case "set-metadata":
		if len(rest) < 3 {
			return errors.New("set-metadata requires bead id and key=value pairs")
		}
		return c.setMetadata(ctx, rest[1], parseKVs(rest[2:]))
	case "close":
		if len(rest) < 2 {
			return errors.New("close requires bead id")
		}
		reason := "doltlite-client manual close"
		if len(rest) > 2 {
			reason = strings.Join(rest[2:], " ")
		}
		return c.closeIssue(ctx, rest[1], reason)
	default:
		usage(fs)
		return fmt.Errorf("unknown command %q", rest[0])
	}
}

func usage(fs *flag.FlagSet) {
	_, _ = fmt.Fprintf(fs.Output(), "usage: doltlite-client [-city PATH] <info|query|exec|show|set-metadata|close> ...\n")
}

func openClient(city, directDBPath string) (*client, error) {
	if strings.TrimSpace(directDBPath) != "" {
		return openDB(filepath.Clean(directDBPath))
	}
	meta, err := readMetadata(city)
	if err != nil {
		return nil, err
	}
	if meta.Backend != "doltlite" {
		return nil, fmt.Errorf("%s is not a doltlite city (backend=%q)", city, meta.Backend)
	}
	dbName := strings.TrimSpace(meta.DoltDatabase)
	if dbName == "" || dbName == "doltlite" {
		dbName = strings.TrimSpace(meta.Database)
	}
	if dbName == "" || dbName == "doltlite" {
		dbName = "hq"
	}
	dbPath := filepath.Join(city, ".beads", "doltlite", dbName+".db")
	return openDB(dbPath)
}

func openDB(dbPath string) (*client, error) {
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?_busy_timeout=10000")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("opening %s: %w", dbPath, err)
	}
	return &client{db: db, path: dbPath}, nil
}

func readMetadata(city string) (metadataFile, error) {
	var meta metadataFile
	data, err := os.ReadFile(filepath.Join(city, ".beads", "metadata.json"))
	if err != nil {
		return meta, err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

func (c *client) info(ctx context.Context) error {
	fmt.Printf("db=%s\n", c.path)
	for _, table := range []string{"issues", "wisps", "labels", "wisp_labels", "dependencies", "wisp_dependencies", "events"} {
		exists, err := c.tableExists(ctx, table)
		if err != nil {
			return err
		}
		if !exists {
			fmt.Printf("table %s: missing\n", table)
			continue
		}
		var count int
		if err := c.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			fmt.Printf("table %s: count error: %v\n", table, err)
			continue
		}
		columns, err := c.columns(ctx, table)
		if err != nil {
			return err
		}
		fmt.Printf("table %s: rows=%d columns=%s\n", table, count, strings.Join(columns, ","))
	}
	return nil
}

func (c *client) query(ctx context.Context, query string, args ...string) error {
	rows, err := c.db.QueryContext(ctx, query, toAny(args)...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	fmt.Println(strings.Join(cols, "\t"))
	count := 0
	for rows.Next() {
		values := make([]sql.NullString, len(cols))
		scan := make([]any, len(cols))
		for i := range values {
			scan[i] = &values[i]
		}
		if err := rows.Scan(scan...); err != nil {
			return err
		}
		out := make([]string, len(cols))
		for i, value := range values {
			if value.Valid {
				out[i] = value.String
			} else {
				out[i] = "NULL"
			}
		}
		fmt.Println(strings.Join(out, "\t"))
		count++
		if count >= 50 {
			fmt.Println("... truncated at 50 rows")
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	fmt.Printf("rows=%d\n", count)
	return nil
}

func (c *client) exec(ctx context.Context, query string, args ...string) error {
	res, err := c.db.ExecContext(ctx, query, toAny(args)...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	fmt.Printf("rows_affected=%d\n", n)
	return nil
}

func (c *client) show(ctx context.Context, id string) error {
	table, err := c.findTable(ctx, id)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("SELECT id, title, status, issue_type, assignee, metadata, updated_at, closed_at FROM %s WHERE id = ?", table)
	return c.query(ctx, query, id)
}

func (c *client) setMetadata(ctx context.Context, id string, kvs map[string]string) error {
	if len(kvs) == 0 {
		return errors.New("no metadata key=value pairs provided")
	}
	table, err := c.findTable(ctx, id)
	if err != nil {
		return err
	}
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	metadata, err := c.loadMetadataTx(ctx, tx, table, id)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(kvs))
	for key, value := range kvs {
		metadata[key] = value
		keys = append(keys, key)
	}
	sort.Strings(keys)
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET metadata = ?, updated_at = ? WHERE id = ?", table), string(data), time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	fmt.Printf("set-metadata table=%s id=%s keys=%s rows=%d\n", table, id, strings.Join(keys, ","), n)
	return nil
}

func (c *client) closeIssue(ctx context.Context, id, reason string) error {
	table, err := c.findTable(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := c.db.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET status = 'closed', closed_at = ?, updated_at = ? WHERE id = ? AND status != 'closed'", table), now, now, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		fmt.Printf("close table=%s id=%s rows=0\n", table, id)
		return nil
	}
	if err := c.insertEvent(ctx, id, "closed", reason, now); err != nil {
		fmt.Printf("close-event-warning id=%s err=%v\n", id, err)
	}
	fmt.Printf("close table=%s id=%s rows=%d\n", table, id, n)
	return nil
}

func (c *client) insertEvent(ctx context.Context, id, eventType, reason, now string) error {
	exists, err := c.tableExists(ctx, "events")
	if err != nil || !exists {
		return err
	}
	cols, err := c.columns(ctx, "events")
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, col := range cols {
		have[col] = true
	}
	if !have["issue_id"] || !have["event_type"] || !have["actor"] || !have["created_at"] {
		return nil
	}
	insertCols := []string{"issue_id", "event_type", "actor", "created_at"}
	insertArgs := []any{id, eventType, "doltlite-client", now}
	if have["id"] {
		insertCols = append([]string{"id"}, insertCols...)
		insertArgs = append([]any{randomUUID()}, insertArgs...)
	}
	if have["comment"] {
		insertCols = append(insertCols, "comment")
		insertArgs = append(insertArgs, reason)
	}
	if have["data"] {
		data, _ := json.Marshal(map[string]string{"reason": reason})
		insertCols = append(insertCols, "data")
		insertArgs = append(insertArgs, string(data))
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(insertCols)), ",")
	_, err = c.db.ExecContext(ctx, "INSERT INTO events ("+strings.Join(insertCols, ",")+") VALUES ("+placeholders+")", insertArgs...)
	return err
}

func (c *client) loadMetadata(ctx context.Context, table, id string) (map[string]any, error) {
	var raw sql.NullString
	if err := c.db.QueryRowContext(ctx, fmt.Sprintf("SELECT metadata FROM %s WHERE id = ?", table), id).Scan(&raw); err != nil {
		return nil, err
	}
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return map[string]any{}, nil
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(raw.String), &metadata); err != nil {
		return nil, fmt.Errorf("parsing metadata for %s: %w", id, err)
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	return metadata, nil
}

func (c *client) loadMetadataTx(ctx context.Context, tx *sql.Tx, table, id string) (map[string]any, error) {
	var raw sql.NullString
	if err := tx.QueryRowContext(ctx, fmt.Sprintf("SELECT metadata FROM %s WHERE id = ?", table), id).Scan(&raw); err != nil {
		return nil, err
	}
	var metadata map[string]any
	if raw.Valid && strings.TrimSpace(raw.String) != "" {
		if err := json.Unmarshal([]byte(raw.String), &metadata); err != nil {
			return nil, fmt.Errorf("parsing metadata for %s: %w", id, err)
		}
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	return metadata, nil
}

func (c *client) findTable(ctx context.Context, id string) (string, error) {
	for _, table := range []string{"wisps", "issues"} {
		exists, err := c.tableExists(ctx, table)
		if err != nil {
			return "", err
		}
		if !exists {
			continue
		}
		var found int
		if err := c.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE id = ?", id).Scan(&found); err != nil {
			return "", err
		}
		if found > 0 {
			return table, nil
		}
	}
	return "", fmt.Errorf("%s: bead not found in issues or wisps", id)
}

func (c *client) tableExists(ctx context.Context, table string) (bool, error) {
	var name string
	err := c.db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *client) columns(ctx context.Context, table string) ([]string, error) {
	rows, err := c.db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var columns []string
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, rows.Err()
}

func parseKVs(args []string) map[string]string {
	kvs := map[string]string{}
	for _, arg := range args {
		key, value, ok := strings.Cut(arg, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		kvs[key] = value
	}
	return kvs
}

func toAny(values []string) []any {
	out := make([]any, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out
}

func randomUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("event-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
