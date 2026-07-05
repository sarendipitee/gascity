package main

import (
	"os"
	"path/filepath"
	"testing"
)

// hs-373vk regression tests for the existence-keyed DoltLite beads-init skip.
// Guards attempt #4 of the house-staff DoltLite backend-plugin cutover:
//   - EXISTING store  -> beads-init must SKIP (the plugin bd can't `init
//     --backend doltlite`; the lifecycle would fail on an existing store).
//   - FRESH store      -> beads-init must NOT be skipped (a blanket skip would
//     silently leave no store — the failure mode we key off store-existence to avoid).

func hs373vkWriteDoltliteCity(t *testing.T, cityDir string) {
	t.Helper()
	content := "[workspace]\nname = \"hs373vk\"\n\n[beads]\nprovider = \"bd\"\nbackend = \"doltlite\"\n"
	if err := os.WriteFile(filepath.Join(cityDir, "city.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// recording canonical DoltLite shim: touches a marker when invoked, exits 0.
func hs373vkWriteRecordingShim(t *testing.T, marker string) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "gc-beads-doltlite-bd.sh") // base name -> canonical provider
	content := "#!/bin/sh\ntouch " + marker + "\nexit 0\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

func hs373vkSeedStore(t *testing.T, cityDir string) {
	t.Helper()
	dl := filepath.Join(cityDir, ".beads", "doltlite")
	if err := os.MkdirAll(dl, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dl, "hs.db"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDoltliteScopeStoreExists_hs373vk(t *testing.T) {
	dir := t.TempDir()
	if doltliteScopeStoreExists(dir) {
		t.Fatal("empty scope: want false")
	}
	hs373vkSeedStore(t, dir)
	if !doltliteScopeStoreExists(dir) {
		t.Fatal("scope with .beads/doltlite/*.db: want true")
	}
}

func TestInitBeadsForDir_doltliteExistingStoreSkipsInit_hs373vk(t *testing.T) {
	cityDir := t.TempDir()
	hs373vkWriteDoltliteCity(t, cityDir)
	marker := filepath.Join(t.TempDir(), "invoked")
	shim := hs373vkWriteRecordingShim(t, marker)
	t.Setenv("GC_BEADS", "exec:"+shim)
	t.Setenv("GC_BEADS_SCOPE_ROOT", cityDir)
	hs373vkSeedStore(t, cityDir) // store already exists -> must skip

	if err := initBeadsForDir(cityDir, cityDir, "hs", "hs"); err != nil {
		t.Fatalf("existing store: want nil, got %v", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("existing store: lifecycle shim was INVOKED — existence-guard failed to skip")
	}
}

func TestInitBeadsForDir_doltliteFreshStoreInvokesInit_hs373vk(t *testing.T) {
	cityDir := t.TempDir()
	hs373vkWriteDoltliteCity(t, cityDir)
	marker := filepath.Join(t.TempDir(), "invoked")
	shim := hs373vkWriteRecordingShim(t, marker)
	t.Setenv("GC_BEADS", "exec:"+shim)
	t.Setenv("GC_BEADS_SCOPE_ROOT", cityDir)
	// NO existing store -> guard must NOT skip; lifecycle MUST run (no silent skip)

	if err := initBeadsForDir(cityDir, cityDir, "hs", "hs"); err != nil {
		t.Fatalf("fresh store: want nil (shim exits 0), got %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("fresh store: lifecycle shim was NOT invoked — guard wrongly skipped a fresh init")
	}
}
