package packlint

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestNoDolt3307InFormulaVarsAndTemplates guards gcy-h0s: formula vars must
// not default to 3307 (the legacy Dolt port that gc-managed cities never use),
// and agent template fragments must not state "port 3307" as a fact. Both
// caused false CRITICAL mail when the managed Dolt server ran on a different
// port (e.g. 55813).
func TestNoDolt3307InFormulaVarsAndTemplates(t *testing.T) {
	root := repoRoot()
	targets := []struct {
		dir     string
		re      *regexp.Regexp
		message string
	}{
		{
			dir:     filepath.Join(root, "examples", "bd", "dolt"),
			re:      regexp.MustCompile(`default\s*=\s*"3307"`),
			message: "formula var has default = \"3307\"; gcy-h0s: use $GC_DOLT_PORT, never a hardcoded port default",
		},
		{
			dir:     filepath.Join(root, "examples", "gastown", "packs", "gastown", "template-fragments"),
			re:      regexp.MustCompile(`port 3307`),
			message: "template fragment states 'port 3307' as a fact; gcy-h0s: reference $GC_DOLT_PORT / gc dolt status instead",
		},
	}

	var hits []string
	for _, target := range targets {
		// On the deploy (`live`) layout the gastown pack template fragments
		// are sourced from the gascity-packs Go module, not this repo, so the
		// template-fragments dir may not exist here. A missing target dir is
		// not a regression — skip it rather than failing the lint.
		if _, statErr := os.Stat(target.dir); os.IsNotExist(statErr) {
			continue
		}
		err := filepath.WalkDir(target.dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".sh" && ext != ".toml" && ext != ".md" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading %s: %w", path, err)
			}
			for lineNo, line := range strings.Split(string(data), "\n") {
				if target.re.MatchString(line) {
					rel, _ := filepath.Rel(root, path)
					hits = append(hits, fmt.Sprintf("%s — %s:%d: %s", target.message, rel, lineNo+1, strings.TrimSpace(line)))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", target.dir, err)
		}
	}

	if len(hits) > 0 {
		t.Fatalf("found %d disallowed hardcoded 3307 reference(s):\n  %s",
			len(hits), strings.Join(hits, "\n  "))
	}
}

// TestNoDolt3307FallbackInScripts guards ga-lsois: the silent
// :=3307 / :-3307 fallback was deleted from pack scripts and formulas
// because nothing listens on 3307 in any gc-managed city. The lint
// fires the build if a regression re-introduces a literal 3307 in the
// authoritative bundled pack sources, except for the explicit allowlist
// below (which slice 2 of ga-lsois — bead ga-nptxjv — will shrink).
func TestNoDolt3307FallbackInScripts(t *testing.T) {
	root := repoRoot()
	packDirs := []string{
		filepath.Join(root, "examples", "bd", "dolt"),
		filepath.Join(root, "internal", "bootstrap", "packs", "core"),
	}

	// Matches GC_DOLT_PORT.*3307 on a single line in source files. The
	// regex is deliberately broad: it catches `:-3307`, `:=3307`,
	// `default = "3307"` on GC_DOLT_PORT lines, and any future variation
	// operators might re-introduce. False positives are easier to
	// allowlist than false negatives are to catch.
	re := regexp.MustCompile(`GC_DOLT_PORT.*3307`)

	var hits []string
	for _, packsDir := range packDirs {
		err := filepath.WalkDir(packsDir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".sh" && ext != ".toml" && ext != ".md" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading %s: %w", path, err)
			}
			for lineNo, line := range strings.Split(string(data), "\n") {
				if re.MatchString(line) {
					rel, _ := filepath.Rel(root, path)
					hits = append(hits, fmt.Sprintf("%s:%d: %s", rel, lineNo+1, strings.TrimSpace(line)))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", packsDir, err)
		}
	}

	if len(hits) > 0 {
		t.Fatalf("found %d disallowed GC_DOLT_PORT/3307 fallback(s); ga-lsois removed this fallback. Re-introduce only by allowlisting in this test (and explaining why):\n  %s",
			len(hits), strings.Join(hits, "\n  "))
	}
}
