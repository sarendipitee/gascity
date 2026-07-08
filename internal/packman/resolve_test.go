package packman

import (
	"errors"
	"strings"
	"testing"
)

func TestResolveVersionLatestMatchingConstraint(t *testing.T) {
	prev := runNetworkGit
	runNetworkGit = func(_, _, _ string, _ ...string) (string, error) {
		return "aaa\trefs/tags/v1.2.0\nbbb\trefs/tags/v1.3.1\nccc\trefs/tags/v2.0.0\n", nil
	}
	t.Cleanup(func() { runNetworkGit = prev })

	got, err := ResolveVersion("", "https://github.com/example/repo", "^1.2")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got.Version != "1.3.1" || got.Commit != "bbb" {
		t.Fatalf("ResolveVersion = %#v", got)
	}
}

func TestResolveVersionSupportsComparators(t *testing.T) {
	prev := runNetworkGit
	runNetworkGit = func(_, _, _ string, _ ...string) (string, error) {
		return "aaa\trefs/tags/v1.2.0\nbbb\trefs/tags/v1.2.5\nccc\trefs/tags/v1.3.0\n", nil
	}
	t.Cleanup(func() { runNetworkGit = prev })

	got, err := ResolveVersion("", "https://github.com/example/repo", ">=1.2.0,<1.3.0")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got.Version != "1.2.5" {
		t.Fatalf("Version = %q, want %q", got.Version, "1.2.5")
	}
}

func TestResolveVersionSupportsSHA(t *testing.T) {
	got, err := ResolveVersion("", "https://github.com/example/repo", "sha:deadbeef")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got.Version != "sha:deadbeef" || got.Commit != "deadbeef" {
		t.Fatalf("ResolveVersion = %#v", got)
	}
}

func TestResolveVersionRedactsUserinfoInError(t *testing.T) {
	prev := runNetworkGit
	runNetworkGit = func(_, _, _ string, _ ...string) (string, error) {
		return "", errors.New("git failed")
	}
	t.Cleanup(func() { runNetworkGit = prev })

	_, err := ResolveVersion("", "https://user:ghp_secret@github.com/example/repo", "^1.0")
	if err == nil {
		t.Fatalf("expected an error from the failing ls-remote")
	}
	if strings.Contains(err.Error(), "ghp_secret") {
		t.Fatalf("error leaked the userinfo token: %v", err)
	}
}

func TestResolveVersionSupportsRefSelector(t *testing.T) {
	prev := runNetworkGit
	var gotArgs []string
	var gotURL string
	runNetworkGit = func(_, url, _ string, args ...string) (string, error) {
		gotURL = url
		gotArgs = append([]string(nil), args...)
		return "abc123\trefs/heads/main\n", nil
	}
	t.Cleanup(func() { runNetworkGit = prev })

	got, err := ResolveVersion("", "https://github.com/example/repo/tree/main/packs/foo", "ref:main")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got.Version != "ref:main" || got.Commit != "abc123" {
		t.Fatalf("ResolveVersion = %#v", got)
	}
	if gotURL != "https://github.com/example/repo.git" {
		t.Fatalf("runNetworkGit url = %q, want normalized clone URL", gotURL)
	}
	wantArgs := []string{
		"ls-remote",
		"https://github.com/example/repo.git",
		"main",
		"refs/heads/main",
		"refs/tags/main",
		"refs/tags/main^{}",
	}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("runGit args = %#v, want %#v", gotArgs, wantArgs)
	}
	for i := range wantArgs {
		if gotArgs[i] != wantArgs[i] {
			t.Fatalf("runGit args = %#v, want %#v", gotArgs, wantArgs)
		}
	}
}

func TestResolveVersionPeelsExplicitAnnotatedTagRef(t *testing.T) {
	prev := runNetworkGit
	var gotArgs []string
	runNetworkGit = func(_, _, _ string, args ...string) (string, error) {
		gotArgs = append([]string(nil), args...)
		return "tagobj\trefs/tags/v1.2.3\ncommit123\trefs/tags/v1.2.3^{}\n", nil
	}
	t.Cleanup(func() { runNetworkGit = prev })

	got, err := ResolveVersion("", "https://github.com/example/repo", "ref:refs/tags/v1.2.3")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got.Commit != "commit123" {
		t.Fatalf("Commit = %q, want peeled commit", got.Commit)
	}
	wantArgs := []string{
		"ls-remote",
		"https://github.com/example/repo",
		"refs/tags/v1.2.3",
		"refs/tags/v1.2.3^{}",
	}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("runGit args = %#v, want %#v", gotArgs, wantArgs)
	}
	for i := range wantArgs {
		if gotArgs[i] != wantArgs[i] {
			t.Fatalf("runGit args = %#v, want %#v", gotArgs, wantArgs)
		}
	}
}

func TestDefaultConstraint(t *testing.T) {
	got, err := DefaultConstraint("1.4.2")
	if err != nil {
		t.Fatalf("DefaultConstraint: %v", err)
	}
	if got != "^1.4" {
		t.Fatalf("DefaultConstraint = %q, want %q", got, "^1.4")
	}
}
