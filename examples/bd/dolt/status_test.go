package dolt_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const statusScript = "commands/status/run.sh"

func TestStatusScriptFallsBackToProbeWhenRuntimePortCannotResolve(t *testing.T) {
	cityPath := t.TempDir()
	packDir := repoRoot(t)
	runtimeDir := filepath.Join(cityPath, ".gc", "runtime", "packs", "dolt")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cityPath, ".gc", "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir scripts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "dolt-state.json"), []byte(`{"running":false,"pid":0,"port":55813,"data_dir":"`+filepath.Join(cityPath, ".beads", "dolt")+`"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write state fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cityPath, ".gc", "scripts", "gc-beads-bd.sh"), []byte("#!/bin/sh\nif [ \"$1\" = \"probe\" ]; then\n  exit 2\nfi\nexit 99\n"), 0o755); err != nil {
		t.Fatalf("write gc-beads-bd stub: %v", err)
	}

	cmd := exec.Command("sh", filepath.Join(packDir, statusScript))
	cmd.Env = append(filteredEnv("GC_CITY_PATH", "GC_PACK_DIR"),
		"GC_CITY_PATH="+cityPath,
		"GC_PACK_DIR="+packDir,
	)
	err := cmd.Run()
	if err == nil {
		t.Fatal("status.sh exited 0, want non-zero for not-running server")
	}
	exitErr := &exec.ExitError{}
	ok := errors.As(err, &exitErr)
	if !ok {
		t.Fatalf("status.sh run error = %v, want ExitError", err)
	}
	if got := exitErr.ExitCode(); got != 1 {
		t.Fatalf("status.sh exit code = %d, want 1", got)
	}
}
