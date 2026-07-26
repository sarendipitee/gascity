package herdr

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gastownhall/gascity/internal/runtime"
)

func TestPlacementUnavailableClassification(t *testing.T) {
	blocked := placementUnavailablef("pane occupied")
	if !isPlacementUnavailable(blocked) {
		t.Fatalf("explicit blocked placement was not classified")
	}
	if isPlacementUnavailable(errors.New("pane occupied")) {
		t.Fatalf("untyped operational error was classified as a recoverable placement collision")
	}
}

func TestStartPrimaryFailureNeverClosesCreatedPane(t *testing.T) {
	p, calls := providerSafetyTestProvider(t, `case "$1 $2" in
"workspace list") printf '%s' '{"result":{"workspaces":[]}}' ;;
"workspace create") printf '%s' '{"result":{"workspace":{"workspace_id":"w1"},"tab":{"tab_id":"t1"},"root_pane":{"pane_id":"p1"}}}' ;;
"tab rename") printf '%s' '{"result":{}}' ;;
"agent start") printf '%s' '{"error":{"code":"launch_failed","message":"invalid launch"}}' >&2; exit 1 ;;
*) printf '%s' '{"result":{}}' ;;
esac
`)

	err := p.Start(context.Background(), "worker", runtime.Config{
		WorkDir: t.TempDir(),
		Command: "omp",
		Env:     map[string]string{"GC_PROVIDER": "omp"},
	})
	if err == nil || !strings.Contains(err.Error(), "launch_failed") {
		t.Fatalf("Start error = %v; want actionable primary launch failure", err)
	}
	if got := readCollisionCalls(t, calls); strings.Contains(got, "pane close") {
		t.Fatalf("failed primary start closed a potentially foreign pane:\n%s", got)
	}
}

func TestStartRecoveryFailureNeverClosesCreatedPanes(t *testing.T) {
	p, calls := providerSafetyTestProvider(t, `case "$1 $2" in
"workspace list") printf '%s' '{"result":{"workspaces":[{"workspace_id":"w1","label":"rig"}]}}' ;;
"tab list") printf '%s' '{"result":{"tabs":[]}}' ;;
"tab create")
  if [ "$6" = "worker" ]; then
    printf '%s' '{"result":{"tab":{"tab_id":"t1"},"root_pane":{"pane_id":"p1"}}}'
  else
    printf '%s' '{"result":{"tab":{"tab_id":"t2"},"root_pane":{"pane_id":"p2"}}}'
  fi ;;
"agent start")
  if [ "$7" = "p1" ]; then
    printf '%s' '{"error":{"code":"agent_pane_busy","message":"claimed"}}' >&2
  else
    printf '%s' '{"error":{"code":"launch_failed","message":"invalid recovery launch"}}' >&2
  fi
  exit 1 ;;
*) printf '%s' '{"result":{}}' ;;
esac
`)

	err := p.Start(context.Background(), "worker", runtime.Config{
		WorkDir: t.TempDir(),
		Command: "omp",
		Env: map[string]string{
			"GC_PROVIDER": "omp",
			"GC_RIG":      "rig",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid recovery launch") {
		t.Fatalf("Start error = %v; want actionable recovery launch failure", err)
	}
	if got := readCollisionCalls(t, calls); strings.Contains(got, "pane close") {
		t.Fatalf("failed recovery start closed a potentially foreign pane:\n%s", got)
	}
}

func providerSafetyTestProvider(t *testing.T, script string) (*Provider, string) {
	t.Helper()
	c, calls := collisionTestClient(t, script)
	home, err := os.MkdirTemp("/tmp", "herdr-test-")
	if err != nil {
		t.Fatalf("create short fake herdr home: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Dir(c.socketPath()), 0o755); err != nil {
		t.Fatalf("create fake herdr socket directory: %v", err)
	}
	listener, err := net.Listen("unix", c.socketPath())
	if err != nil {
		t.Fatalf("listen fake herdr server: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	p := New("test", t.TempDir(), c.cityRoot, 0, 0)
	p.c = c
	return p, calls
}
