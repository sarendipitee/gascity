package beads

import (
	"path/filepath"
	"testing"
)

func TestResolveBackendPluginCommandUsesExplicitGasCityCommand(t *testing.T) {
	command, args, err := resolveBackendPluginCommand(backendPluginMetadata{
		GasCityBackendCommand: "/tmp/gc-backend-plugin",
		GasCityBackendArgs:    []string{"serve", "--debug"},
		BackendPluginCommand:  "/tmp/bd-backend-doltlite",
	})
	if err != nil {
		t.Fatalf("resolveBackendPluginCommand: %v", err)
	}
	if command != "/tmp/gc-backend-plugin" {
		t.Fatalf("command = %q, want explicit gascity command", command)
	}
	if len(args) != 2 || args[0] != "serve" || args[1] != "--debug" {
		t.Fatalf("args = %#v, want explicit args", args)
	}
}

func TestResolveBackendPluginCommandInfersSiblingFromBackendPlugin(t *testing.T) {
	command, args, err := resolveBackendPluginCommand(backendPluginMetadata{
		BackendPluginCommand: "/opt/beads/bin/bd-backend-doltlite",
	})
	if err != nil {
		t.Fatalf("resolveBackendPluginCommand: %v", err)
	}
	want := filepath.Join("/opt/beads/bin", "gc-doltlite-fastpath")
	if command != want {
		t.Fatalf("command = %q, want %q", command, want)
	}
	if len(args) != 1 || args[0] != "serve" {
		t.Fatalf("args = %#v, want [serve]", args)
	}
}

func TestBackendPluginIssueRoundTripPreservesGasCityFields(t *testing.T) {
	priority := 1
	input := Bead{
		ID:          "gc-1",
		Title:       "work",
		Status:      "in_progress",
		Type:        "task",
		Priority:    &priority,
		Assignee:    "agent",
		From:        "mayor",
		ParentID:    "gc-parent",
		Description: "details",
		Labels:      []string{"one", "one", "two"},
		Metadata:    map[string]string{"k": "v"},
		Ephemeral:   true,
	}
	issue := beadToPluginIssue(input)
	output := pluginIssueToBead(issue)
	output.Dependencies = nil
	output.ParentID = ""
	if output.ID != input.ID || output.Title != input.Title || output.Status != input.Status || output.Type != input.Type {
		t.Fatalf("round trip identity = %#v, want %#v", output, input)
	}
	if output.Priority == nil || *output.Priority != priority {
		t.Fatalf("priority = %#v, want %d", output.Priority, priority)
	}
	if output.From != "mayor" || output.Metadata["from"] != "mayor" || output.Metadata["k"] != "v" {
		t.Fatalf("metadata/from = %#v / %q", output.Metadata, output.From)
	}
	if len(output.Labels) != 2 || output.Labels[0] != "one" || output.Labels[1] != "two" {
		t.Fatalf("labels = %#v, want compact labels", output.Labels)
	}
	if !output.Ephemeral {
		t.Fatalf("ephemeral = false, want true")
	}
}

func TestCreateDependenciesForBeadAddsParentAndNeeds(t *testing.T) {
	deps := createDependenciesForBead("gc-child", Bead{
		ParentID: "gc-parent",
		Needs:    []string{"blocks:gc-blocker", "gc-default-blocker"},
	})
	if len(deps) != 3 {
		t.Fatalf("deps = %#v, want 3", deps)
	}
	want := []backendPluginDependency{
		{IssueID: "gc-child", DependsOnID: "gc-parent", Type: "parent-child"},
		{IssueID: "gc-child", DependsOnID: "gc-blocker", Type: "blocks"},
		{IssueID: "gc-child", DependsOnID: "gc-default-blocker", Type: "blocks"},
	}
	for i := range want {
		if deps[i].IssueID != want[i].IssueID || deps[i].DependsOnID != want[i].DependsOnID || deps[i].Type != want[i].Type {
			t.Fatalf("deps[%d] = %#v, want %#v", i, deps[i], want[i])
		}
	}
}
