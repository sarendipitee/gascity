package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gastownhall/gascity/internal/beads/contract"
	"github.com/gastownhall/gascity/internal/fsys"
)

func TestBeadsBackendCapabilitiesConformance(t *testing.T) {
	tests := []struct {
		name    string
		backend BeadsBackend
	}{
		{name: "dolt", backend: resolveBeadsBackendName("dolt")},
		{name: "doltlite", backend: resolveBeadsBackendName("doltlite")},
		{name: "postgres", backend: resolveBeadsBackendName("postgres")},
		{name: "unknown_external", backend: resolveBeadsBackendName("custom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := tt.backend.Capabilities()
			if got := caps.ManagedServer; got != tt.backend.NeedsManagedServer() {
				t.Fatalf("ManagedServer = %v, NeedsManagedServer = %v", got, tt.backend.NeedsManagedServer())
			}
			if got := caps.DoltBinary; got != tt.backend.NeedsDoltBinary() {
				t.Fatalf("DoltBinary = %v, NeedsDoltBinary = %v", got, tt.backend.NeedsDoltBinary())
			}
			if got := caps.BeadHooks; got != tt.backend.NeedsBeadHooks() {
				t.Fatalf("BeadHooks = %v, NeedsBeadHooks = %v", got, tt.backend.NeedsBeadHooks())
			}
			if got := caps.DoltDoctorChecks; got != tt.backend.NeedsDoltDoctorChecks() {
				t.Fatalf("DoltDoctorChecks = %v, NeedsDoltDoctorChecks = %v", got, tt.backend.NeedsDoltDoctorChecks())
			}
			if caps.OptimizedLocalStore && caps.OptimizedStoreName == "" {
				t.Fatalf("OptimizedLocalStore=true but OptimizedStoreName is empty")
			}
			if !caps.OptimizedLocalStore && caps.OptimizedStoreName != "" {
				t.Fatalf("OptimizedLocalStore=false but OptimizedStoreName = %q", caps.OptimizedStoreName)
			}
		})
	}
}

func TestBeadsBackendOptimizedStoreCapabilityConformance(t *testing.T) {
	if resolveBeadsBackendName("dolt").Capabilities().OptimizedLocalStore {
		t.Fatal("dolt backend must not advertise Gas City optimized local store")
	}
	if !resolveBeadsBackendName("doltlite").Capabilities().OptimizedLocalStore {
		t.Fatal("doltlite backend should advertise Gas City optimized local store capability")
	}
	if resolveBeadsBackendName("postgres").Capabilities().OptimizedLocalStore {
		t.Fatal("postgres backend must not advertise Gas City optimized local store")
	}
}

func TestScopeNeedsDoltDoctorChecksFollowsBackendCapability(t *testing.T) {
	clearInheritedBeadsEnv(t)

	doltCity := writeBackendCity(t, "dolt")
	if !scopeNeedsDoltDoctorChecks(doltCity, doltCity) {
		t.Fatal("dolt city should need built-in Dolt doctor checks")
	}

	doltliteCity := writeBackendCity(t, "doltlite")
	if scopeNeedsDoltDoctorChecks(doltliteCity, doltliteCity) {
		t.Fatal("doltlite city should use pack-managed doctor checks")
	}

	inheritedRig := filepath.Join(doltliteCity, "rigs", "dog")
	if err := os.MkdirAll(filepath.Join(inheritedRig, ".beads"), 0o755); err != nil {
		t.Fatalf("mkdir inherited rig: %v", err)
	}
	if scopeNeedsDoltDoctorChecks(doltliteCity, inheritedRig) {
		t.Fatal("rig without backend override should inherit doltlite doctor behavior")
	}

	explicitRig := filepath.Join(doltliteCity, "rigs", "ops")
	if err := os.MkdirAll(filepath.Join(explicitRig, ".beads"), 0o755); err != nil {
		t.Fatalf("mkdir explicit rig: %v", err)
	}
	if _, err := contract.EnsureCanonicalConfig(fsys.OSFS{}, filepath.Join(explicitRig, ".beads", "config.yaml"), contract.ConfigState{
		IssuePrefix:    "ops",
		EndpointOrigin: contract.EndpointOriginExplicit,
		EndpointStatus: contract.EndpointStatusVerified,
		DoltHost:       "db.example.test",
		DoltPort:       "3306",
		DoltUser:       "bd",
	}); err != nil {
		t.Fatalf("write explicit rig config: %v", err)
	}
	if !scopeNeedsDoltDoctorChecks(doltliteCity, explicitRig) {
		t.Fatal("explicit Dolt rig under doltlite city should need built-in Dolt doctor checks")
	}
}

func writeBackendCity(t *testing.T, backend string) string {
	t.Helper()

	cityDir := t.TempDir()
	data := []byte("[workspace]\nname = \"demo\"\n\n[beads]\nprovider = \"bd\"\nbackend = \"" + backend + "\"\n")
	if err := os.WriteFile(filepath.Join(cityDir, "city.toml"), data, 0o644); err != nil {
		t.Fatalf("write city.toml: %v", err)
	}
	return cityDir
}
