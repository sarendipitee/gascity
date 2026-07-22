package herdr

import "testing"

// TestSeedMetaFromEnv pins the start-time metadata seed. Herdr has no native
// session environment, so Start mirrors cfg.Env into its metadata sidecar;
// ownership keys must be readable as soon as the agent exists and the rest of
// the environment follows tmux's GetMeta contract.
func TestSeedMetaFromEnv(t *testing.T) {
	p := New("gctest-identity-meta", t.TempDir(), t.TempDir(), 0)
	if err := p.seedMetaFromEnv("canary", map[string]string{
		"GC_SESSION_ID":     "gm-abc123",
		"GC_INSTANCE_TOKEN": "tok-1",
		"GC_RUNTIME_EPOCH":  "3",
		"GC_CITY":           "not-an-identity-key",
	}); err != nil {
		t.Fatalf("seedMetaFromEnv: %v", err)
	}
	for key, want := range map[string]string{
		"GC_SESSION_ID":     "gm-abc123",
		"GC_INSTANCE_TOKEN": "tok-1",
		"GC_RUNTIME_EPOCH":  "3",
		"GC_CITY":           "not-an-identity-key",
	} {
		got, err := p.GetMeta("canary", key)
		if err != nil {
			t.Fatalf("GetMeta(%s): %v", key, err)
		}
		if got != want {
			t.Errorf("GetMeta(%s) = %q, want %q", key, got, want)
		}
	}

	if err := p.seedMetaFromEnv("empty", map[string]string{"GC_SESSION_ID": ""}); err != nil {
		t.Fatalf("seedMetaFromEnv empty: %v", err)
	}
	if got, _ := p.GetMeta("empty", "GC_SESSION_ID"); got != "" {
		t.Errorf("empty env value stamped as %q, want unstamped", got)
	}
}
