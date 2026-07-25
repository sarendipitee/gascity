package herdr

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gastownhall/gascity/internal/runtime"
)

func TestStartAgentUsesHerdr075PaneContract(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls")
	bin := filepath.Join(dir, "herdr")
	script := `#!/bin/sh
shift 2
printf '%s\n' "$*" >> "$HERDR_TEST_LOG"
case "$1 $2" in
"workspace list") printf '{"result":{"workspaces":[]}}' ;;
"workspace create") printf '{"result":{"workspace":{"workspace_id":"w1"},"tab":{"tab_id":"t1"},"root_pane":{"pane_id":"p1"}}}' ;;
"tab rename") printf '{"result":{}}' ;;
"agent start") printf '{"result":{"agent":{"name":"worker","pane_id":"p1"}}}' ;;
*) printf '{"result":{}}' ;;
esac
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_TEST_LOG", logPath)
	c := newClient("test", dir)
	c.bin = bin

	tabID, paneID, err := c.ensurePlacement(context.Background(), "rig", "worker", "/work", map[string]string{"Z": "last", "A": "first"})
	if err != nil {
		t.Fatalf("ensurePlacement: %v", err)
	}
	if tabID != "t1" || paneID != "p1" {
		t.Fatalf("placement = %q, %q; want t1, p1", tabID, paneID)
	}
	if _, err := c.startAgent(context.Background(), "worker", "omp", paneID, []string{"omp", "--model", "fast"}); err != nil {
		t.Fatalf("startAgent: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{
		"workspace list",
		"workspace create --label rig --no-focus --cwd /work --env A=first --env Z=last",
		"tab rename t1 worker",
		"agent start worker --kind omp --pane p1 -- --model fast",
	}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("herdr calls = %#v, want %#v", lines, want)
	}
}

func TestHerdrAgentLaunchUsesProviderFamilyAndCommandArgs(t *testing.T) {
	kind, argv, err := herdrAgentLaunch(runtime.Config{
		Command: "omp --model 'gpt 5'",
		Env:     map[string]string{"GC_PROVIDER": "omp"},
	})
	if err != nil {
		t.Fatalf("herdrAgentLaunch: %v", err)
	}
	if kind != "omp" || !reflect.DeepEqual(argv, []string{"omp", "--model", "gpt 5"}) {
		t.Fatalf("launch = %q, %#v", kind, argv)
	}
}
