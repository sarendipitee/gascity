package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessBuildCheckForCityMatchesBeadsDoltliteStamp(t *testing.T) {
	cityPath := t.TempDir()
	binaryPath := filepath.Join(cityPath, "gc")
	if err := os.WriteFile(binaryPath, []byte("gc binary\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	sha, err := fileSHA256(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	writeBeadsDoltliteGCBuildStamp(t, cityPath, beadsDoltliteBuildStamp{
		SchemaVersion: "1",
		Pack:          "beads-doltlite",
		Target:        "gc",
		BuiltAt:       "2026-06-07T00:00:00Z",
		BinaryPath:    binaryPath,
		SHA256:        sha,
	})

	got := processBuildCheckForCity(cityPath, binaryPath)
	if got == nil {
		t.Fatal("processBuildCheckForCity returned nil")
	}
	if got.Status != "matched" {
		t.Fatalf("Status = %q, want matched; got %+v", got.Status, got)
	}
	if got.Matches == nil || !*got.Matches {
		t.Fatalf("Matches = %#v, want true", got.Matches)
	}
	if suffix := processDetailsSuffix(binaryPath, got); !strings.Contains(suffix, "build matches beads-doltlite stamp") {
		t.Fatalf("suffix = %q, want build match text", suffix)
	}
}

func TestProcessBuildCheckForCityReportsMismatch(t *testing.T) {
	cityPath := t.TempDir()
	binaryPath := filepath.Join(cityPath, "gc")
	if err := os.WriteFile(binaryPath, []byte("new gc binary\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeBeadsDoltliteGCBuildStamp(t, cityPath, beadsDoltliteBuildStamp{
		SchemaVersion: "1",
		Pack:          "beads-doltlite",
		Target:        "gc",
		BuiltAt:       "2026-06-07T00:00:00Z",
		BinaryPath:    binaryPath,
		SHA256:        "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	})

	got := processBuildCheckForCity(cityPath, binaryPath)
	if got == nil {
		t.Fatal("processBuildCheckForCity returned nil")
	}
	if got.Status != "mismatch" {
		t.Fatalf("Status = %q, want mismatch; got %+v", got.Status, got)
	}
	if got.Matches == nil || *got.Matches {
		t.Fatalf("Matches = %#v, want false", got.Matches)
	}
	if suffix := processDetailsSuffix(binaryPath, got); !strings.Contains(suffix, "build mismatch beads-doltlite stamp expected 0123456789ab") {
		t.Fatalf("suffix = %q, want mismatch text with shortened expected sha", suffix)
	}
}

func writeBeadsDoltliteGCBuildStamp(t *testing.T, cityPath string, stamp beadsDoltliteBuildStamp) {
	t.Helper()
	stampPath := beadsDoltliteGCBuildStampPath(cityPath)
	if err := os.MkdirAll(filepath.Dir(stampPath), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(stamp)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stampPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
