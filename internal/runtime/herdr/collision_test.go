package herdr

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func collisionTestClient(t *testing.T, script string) (*client, string) {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls")
	bin := filepath.Join(dir, "herdr")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nshift 2\nprintf '%s\\n' \"$*\" >> \"$HERDR_TEST_LOG\"\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_TEST_LOG", logPath)
	c := newClient("test", dir)
	c.bin = bin
	return c, logPath
}

func readCollisionCalls(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRunClassifiesAgentPaneBusyFromStderr(t *testing.T) {
	c, _ := collisionTestClient(t, `printf '%s' '{"error":{"code":"agent_pane_busy","message":"occupied"}}' >&2
exit 1
`)
	_, err := c.startAgent(context.Background(), "worker", "omp", "p1", nil)
	if !isHerdrError(err, "agent_pane_busy") {
		t.Fatalf("error = %v; want typed agent_pane_busy", err)
	}
	var herr *herdrError
	if !errors.As(err, &herr) || herr.Message != "occupied" {
		t.Fatalf("typed error = %#v; want occupied message", herr)
	}
}

func TestEnsurePlacementReusesOnlyUniqueUnownedShellPane(t *testing.T) {
	c, calls := collisionTestClient(t, `case "$1 $2" in
"workspace list") printf '%s' '{"result":{"workspaces":[{"workspace_id":"w1","label":"rig"}]}}' ;;
"tab list") printf '%s' '{"result":{"tabs":[{"tab_id":"t1","label":"worker"}]}}' ;;
"pane list") printf '%s' '{"result":{"panes":[{"pane_id":"p1","tab_id":"t1","workspace_id":"w1","agent":null,"agent_session":null},{"pane_id":"other","tab_id":"other-tab","workspace_id":"w1","agent":"omp"}]}}' ;;
"pane process-info") printf '%s' '{"result":{"process_info":{"pane_id":"p1","shell_pid":10,"foreground_processes":[{"pid":10,"name":"bash","argv":["bash"],"cwd":"/work"}]}}}' ;;
esac
`)
	placed, err := c.ensurePlacement(context.Background(), "rig", "worker", "/work", nil)
	if err != nil {
		t.Fatal(err)
	}
	if placed.WorkspaceID != "w1" || placed.TabID != "t1" || placed.PaneID != "p1" || placed.CreatedPane {
		t.Fatalf("placement = %+v; want reused w1/t1/p1", placed)
	}
	if got := readCollisionCalls(t, calls); strings.Contains(got, "pane close") || strings.Contains(got, "tab create") {
		t.Fatalf("clean reuse mutated placement:\n%s", got)
	}
}

func TestEnsurePlacementPreservesForeignAndAmbiguousPanes(t *testing.T) {
	tests := []struct {
		name  string
		panes string
		want  string
	}{
		{"owned", `[{"pane_id":"p1","tab_id":"t1","workspace_id":"w1","agent":"omp","agent_session":{"agent":"omp","kind":"path","source":"herdr:omp","value":"foreign.jsonl"}}]`, "agent=omp"},
		{"multiple", `[{"pane_id":"p1","tab_id":"t1","workspace_id":"w1","agent":null,"agent_session":null},{"pane_id":"p2","tab_id":"t1","workspace_id":"w1","agent":null,"agent_session":null}]`, "pane_count=2"},
		{"non-shell", `[{"pane_id":"p1","tab_id":"t1","workspace_id":"w1","agent":null,"agent_session":null}]`, "foreground"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			process := `{"result":{"process_info":{"shell_pid":10,"foreground_processes":[{"pid":11,"name":"vim","argv":["vim"],"cwd":"/work"}]}}}`
			c, calls := collisionTestClient(t, `case "$1 $2" in
"workspace list") printf '%s' '{"result":{"workspaces":[{"workspace_id":"w1","label":"rig"}]}}' ;;
"tab list") printf '%s' '{"result":{"tabs":[{"tab_id":"t1","label":"worker"}]}}' ;;
"pane list") printf '%s' '{"result":{"panes":`+tt.panes+`}}' ;;
"pane process-info") printf '%s' '`+process+`' ;;
esac
`)
			_, err := c.ensurePlacement(context.Background(), "rig", "worker", "/work", nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v; want diagnostic containing %q", err, tt.want)
			}
			if got := readCollisionCalls(t, calls); strings.Contains(got, "pane close") || strings.Contains(got, "tab create") {
				t.Fatalf("blocked placement was mutated:\n%s", got)
			}
		})
	}
}

func TestEnsurePlacementRejectsDuplicateExactLabels(t *testing.T) {
	c, _ := collisionTestClient(t, `case "$1 $2" in
"workspace list") printf '%s' '{"result":{"workspaces":[{"workspace_id":"w1","label":"rig"},{"workspace_id":"w2","label":"rig"}]}}' ;;
esac
`)
	_, err := c.ensurePlacement(context.Background(), "rig", "worker", "/work", nil)
	if err == nil || !strings.Contains(err.Error(), "workspace_ids=[w1 w2]") {
		t.Fatalf("error = %v; want duplicate identities", err)
	}
}

func TestRecoveryLabelIsDeterministicBoundedAndReusable(t *testing.T) {
	label := recoveryTabLabel("a-very-long-agent-tab-label-that-must-be-bounded")
	if label != recoveryTabLabel("a-very-long-agent-tab-label-that-must-be-bounded") || len(label) > 32 {
		t.Fatalf("recovery label = %q (len %d); want deterministic <=32", label, len(label))
	}
	c, calls := collisionTestClient(t, `case "$1 $2" in
"workspace list") printf '%s' '{"result":{"workspaces":[{"workspace_id":"w1","label":"rig"}]}}' ;;
"tab list") printf '%s' '{"result":{"tabs":[{"tab_id":"tr","label":"`+label+`"}]}}' ;;
"pane list") printf '%s' '{"result":{"panes":[{"pane_id":"pr","tab_id":"tr","workspace_id":"w1","agent":null,"agent_session":null}]}}' ;;
"pane process-info") printf '%s' '{"result":{"process_info":{"shell_pid":10,"foreground_processes":[{"pid":10,"name":"zsh","argv":["zsh"],"cwd":"/work"}]}}}' ;;
esac
`)
	for i := 0; i < 2; i++ {
		placed, err := c.ensurePlacement(context.Background(), "rig", label, "/work", nil)
		if err != nil || placed.PaneID != "pr" || placed.CreatedPane {
			t.Fatalf("attempt %d placement=%+v error=%v", i, placed, err)
		}
	}
	if got := readCollisionCalls(t, calls); strings.Contains(got, "tab create") || strings.Contains(got, "pane close") {
		t.Fatalf("recovery reuse created or deleted placement:\n%s", got)
	}
}

func TestRecoveryStartRetryIsBoundedByCaller(t *testing.T) {
	c, calls := collisionTestClient(t, `printf '%s' '{"error":{"code":"agent_pane_busy","message":"still occupied"}}' >&2
exit 1
`)
	_, err := c.startAgent(context.Background(), "worker", "omp", "recovery-pane", nil)
	if !isHerdrError(err, "agent_pane_busy") {
		t.Fatalf("error = %v; want typed busy", err)
	}
	if got := strings.Count(readCollisionCalls(t, calls), "agent start"); got != 1 {
		t.Fatalf("agent start calls = %d; want exactly one recovery attempt", got)
	}
}
