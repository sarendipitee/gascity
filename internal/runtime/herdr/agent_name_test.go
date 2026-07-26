package herdr

import "testing"

func TestHerdrAgentNameConformsToHerdrLimit(t *testing.T) {
	const runtimeName = "gc__design-implementation-reviewer-ci-h4pq3d"

	got := herdrAgentName(runtimeName)
	if got == runtimeName {
		t.Fatal("long runtime name was not normalized")
	}
	if len(got) > 32 || !isHerdrAgentName(got) {
		t.Fatalf("normalized name %q is not Herdr-compatible", got)
	}
	if got != herdrAgentName(runtimeName) {
		t.Fatalf("normalization is not deterministic: %q", got)
	}
	if got == herdrAgentName("gc__design-implementation-reviewer-ci-7tpl5") {
		t.Fatalf("distinct runtime names collided: %q", got)
	}
	if got := herdrAgentName("worker_1"); got != "worker_1" {
		t.Fatalf("valid name was rewritten: %q", got)
	}
}
