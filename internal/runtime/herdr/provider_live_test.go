package herdr

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/gastownhall/gascity/internal/runtime"
)

// TestProviderLive drives the herdr Provider against a real herdr binary in an
// isolated session. Skipped when herdr is unavailable or in -short mode.
func TestProviderLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live herdr test in -short mode")
	}
	if _, err := exec.LookPath("herdr"); err != nil {
		t.Skip("herdr not installed")
	}

	p := New("gctest-live", t.TempDir(), t.TempDir(), 0, 0)
	_ = p.Stop("smoke") // clear any leftover from a crashed prior run
	t.Cleanup(func() { _ = p.Stop("smoke"); _ = p.TeardownServer() })

	ctx := context.Background()
	cfg := runtime.Config{
		WorkDir: t.TempDir(),
		Command: "omp",
		Env:     map[string]string{"GC_SESSION_ID": "gctest-live-session", "GC_PROVIDER": "omp"},
	}
	if err := p.Start(ctx, "smoke", cfg); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Start must persist GC_SESSION_ID to the meta sidecar (tmux parity):
	// ProcessAlive's session-scoped tree-walk widening has nothing to read
	// without it.
	if v, err := p.GetMeta("smoke", "GC_SESSION_ID"); err != nil || v != "gctest-live-session" {
		t.Errorf("GetMeta(GC_SESSION_ID) = %q, %v; want %q, nil", v, err, "gctest-live-session")
	}

	if !p.IsRunning("smoke") {
		t.Error("IsRunning = false after Start")
	}
	if names, err := p.ListRunning("smo"); err != nil || len(names) != 1 || names[0] != "smoke" {
		t.Errorf("ListRunning(smo) = %v, %v; want [smoke]", names, err)
	}

	// Start returning and IsRunning above prove Herdr detected the supported
	// agent in the requested pane and registered it by name.

	// Metadata sidecar roundtrip.
	if err := p.SetMeta("smoke", "drain", "1"); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	if v, err := p.GetMeta("smoke", "drain"); err != nil || v != "1" {
		t.Errorf("GetMeta(drain) = %q, %v; want 1", v, err)
	}
	if v, err := p.GetMeta("smoke", "absent"); err != nil || v != "" {
		t.Errorf("GetMeta(absent) = %q, %v; want empty,nil", v, err)
	}

	// Stop → no longer running.
	if err := p.Stop("smoke"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	for i := 0; i < 10 && p.IsRunning("smoke"); i++ {
		time.Sleep(200 * time.Millisecond)
	}
	if p.IsRunning("smoke") {
		t.Error("IsRunning = true after Stop")
	}
}

func TestProviderLiveLongIdentifier(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live herdr test in -short mode")
	}
	if _, err := exec.LookPath("herdr"); err != nil {
		t.Skip("herdr not installed")
	}

	const name = "gc__design-implementation-reviewer-ci-h4pq3d"
	p := New("gctest-long-identifier", t.TempDir(), t.TempDir(), 0, 0)
	_ = p.Stop(name)
	t.Cleanup(func() { _ = p.Stop(name); _ = p.TeardownServer() })

	cfg := runtime.Config{
		WorkDir: t.TempDir(),
		Command: "omp",
		Env: map[string]string{
			"GC_PROVIDER":   "omp",
			"GC_SESSION_ID": "gctest-long-identifier-session",
		},
	}
	if err := p.Start(context.Background(), name, cfg); err != nil {
		t.Fatalf("Start(%q): %v", name, err)
	}
	if !p.IsRunning(name) {
		t.Fatalf("IsRunning(%q) = false after Start", name)
	}
	if names, err := p.ListRunning("gc__design"); err != nil || len(names) != 1 || names[0] != name {
		t.Errorf("ListRunning = %v, %v; want [%q], nil", names, err, name)
	}
}
