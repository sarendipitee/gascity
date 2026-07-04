package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreCommitFormatterPreservesFileMode(t *testing.T) {
	repoRoot := repoRoot(t)
	binDir := t.TempDir()
	fakeLint := filepath.Join(binDir, "golangci-lint")
	writeExecutable(t, fakeLint, `#!/usr/bin/env bash
set -euo pipefail
if [ "$#" -ne 2 ] || [ "$1" != "fmt" ] || [ "$2" != "--stdin" ]; then
  echo "unexpected golangci-lint args: $*" >&2
  exit 2
fi
cat
printf '\n'
`)

	source := filepath.Join(t.TempDir(), "needs_format.go")
	if err := os.WriteFile(source, []byte("package main"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	cmd := exec.Command(filepath.Join(repoRoot, "scripts", "precommit-format-staged-go"))
	cmd.Dir = repoRoot
	cmd.Env = []string{
		"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + t.TempDir(),
		"TMPDIR=" + t.TempDir(),
	}
	cmd.Stdin = strings.NewReader(source + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("precommit formatter failed: %v\n%s", err, out)
	}

	info, err := os.Stat(source)
	if err != nil {
		t.Fatalf("stat formatted source: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("formatted source mode = %o, want 644", got)
	}
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read formatted source: %v", err)
	}
	if string(content) != "package main\n" {
		t.Fatalf("formatted content = %q, want package main with newline", content)
	}
}

func TestTestFastParallelUsesSanitizedEnvironment(t *testing.T) {
	repoRoot := repoRoot(t)
	cmd := exec.Command("make", "-n", "test-fast-parallel")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n test-fast-parallel failed: %v\n%s", err, out)
	}
	command := string(out)
	if !strings.Contains(command, "env -i") {
		t.Fatalf("test-fast-parallel recipe should use TEST_ENV env -i wrapper:\n%s", command)
	}
	if !strings.Contains(command, "./scripts/test-local-parallel fast") {
		t.Fatalf("test-fast-parallel recipe should still dispatch the sharded fast runner:\n%s", command)
	}
}

func TestNativeDoltliteBeadsTargetRunsTaggedSuite(t *testing.T) {
	repoRoot := repoRoot(t)
	cmd := exec.Command("make", "-n", "test-native-doltlite-beads")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n test-native-doltlite-beads failed: %v\n%s", err, out)
	}
	command := string(out)
	for _, want := range []string{
		"CGO_ENABLED=1",
		"DOLTLITE_LIB=",
		"GC_DOLTLITE_LIB=",
			"LD_LIBRARY_PATH=",
			"CGO_LDFLAGS=\"-L$lib_dir",
			"scripts/test-native-doltlite",
		} {
			if !strings.Contains(command, want) {
				t.Fatalf("test-native-doltlite-beads recipe missing %q:\n%s", want, command)
			}
		}
	for _, banned := range []string{
		"CGO_ENABLED=0",
		"modernc",
		"cannot find -ldoltlite",
	} {
		if strings.Contains(command, banned) {
			t.Fatalf("test-native-doltlite-beads recipe must not contain %q:\n%s", banned, command)
		}
	}

	runner, err := os.ReadFile(filepath.Join(repoRoot, "scripts", "test-native-doltlite"))
	if err != nil {
		t.Fatalf("read native DoltLite test runner: %v", err)
	}
	runnerText := string(runner)
	for _, want := range []string{
		"git ls-files '*_test.go'",
		"-tags gascity_doltlite_lib",
		"go test",
	} {
		if !strings.Contains(runnerText, want) {
			t.Fatalf("native DoltLite test runner missing %q:\n%s", want, runnerText)
		}
	}

	listCmd := exec.Command(filepath.Join(repoRoot, "scripts", "test-native-doltlite"), "--list")
	listCmd.Dir = repoRoot
	listOut, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list native DoltLite tests: %v\n%s", err, listOut)
	}
	list := string(listOut)
	for _, want := range []string{
		"./internal/beads\t",
		"./cmd/gc\tTestDoltliteReadFastPathEnabled",
	} {
		if !strings.Contains(list, want) {
			t.Fatalf("native DoltLite test discovery missing %q:\n%s", want, list)
		}
	}
	for _, unwanted := range []string{
		"/.gc/",
		"/workspaces/",
		"./gc-",
	} {
		if strings.Contains(list, unwanted) {
			t.Fatalf("native DoltLite test discovery included workspace path %q:\n%s", unwanted, list)
		}
	}
}

func TestLocalParallelAllowlistIncludesObservableEnv(t *testing.T) {
	repoRoot := repoRoot(t)
	script, err := os.ReadFile(filepath.Join(repoRoot, "scripts", "test-local-parallel"))
	if err != nil {
		t.Fatalf("read test-local-parallel: %v", err)
	}
	content := string(script)
	for _, key := range []string{"OBSERVABLE_TEST_LOG", "OBSERVABLE_FAILURE_LINES"} {
		if !strings.Contains(content, key+"=") {
			t.Fatalf("test-local-parallel job env should pass through %s", key)
		}
	}
	for _, key := range []string{"GC_CITY", "GC_HOME", "GC_SESSION_ID"} {
		if strings.Contains(content, key+"=") {
			t.Fatalf("test-local-parallel job env must not pass through live session env %s", key)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Dir(wd)
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}
