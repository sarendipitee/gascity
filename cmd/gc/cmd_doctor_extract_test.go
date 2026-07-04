package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gastownhall/gascity/internal/config"
	"github.com/gastownhall/gascity/internal/doctor"
)

func TestBuildDoctorChecks_NameSetUnchanged(t *testing.T) {
	cityDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(cityDir, "city.toml"), []byte("[workspace]\nname = \"demo\"\n"), 0o644); err != nil {
		t.Fatalf("write city.toml: %v", err)
	}
	t.Setenv("GC_DOLT", "skip")
	cfg := &config.City{Workspace: config.Workspace{Name: "demo"}}

	checks := buildDoctorChecks(cityDir, cfg, nil, buildDoctorChecksOpts{
		ControllerRunning:    false,
		SkipCityDoltCheck:    true,
		SkipManagedDoltCheck: true,
	})
	names := doctorCheckNames(checks)

	data, err := os.ReadFile(filepath.Join("testdata", "doctor_check_names.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	want := strings.TrimSpace(string(data))
	got := strings.Join(names, "\n")
	if got != want {
		t.Fatalf("doctor check names changed\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestBuildDoctorChecksRegistersNamedAlwaysMinConflictCheck(t *testing.T) {
	cityDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(cityDir, "city.toml"), []byte("[workspace]\nname = \"demo\"\n"), 0o644); err != nil {
		t.Fatalf("write city.toml: %v", err)
	}
	t.Setenv("GC_DOLT", "skip")
	cfg := &config.City{Workspace: config.Workspace{Name: "demo"}}

	names := doctorCheckNames(buildDoctorChecks(cityDir, cfg, nil, buildDoctorChecksOpts{
		ControllerRunning:    false,
		SkipCityDoltCheck:    true,
		SkipManagedDoltCheck: true,
	}))

	formulaRequirements := doctorCheckIndex(names, "formula-requirements")
	if formulaRequirements < 0 {
		t.Fatalf("formula-requirements check missing: %v", names)
	}
	got := doctorCheckIndex(names, "named-always-min-conflict")
	if got != formulaRequirements+1 {
		t.Fatalf("named-always-min-conflict index = %d, want immediately after formula-requirements at %d; names=%v", got, formulaRequirements, names)
	}
}

func TestBuildDoctorChecksSkipsNamedAlwaysMinConflictCheckWithoutConfig(t *testing.T) {
	cityDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(cityDir, "city.toml"), []byte("[workspace]\nname = \"demo\"\n"), 0o644); err != nil {
		t.Fatalf("write city.toml: %v", err)
	}
	t.Setenv("GC_DOLT", "skip")

	tests := []struct {
		name   string
		cfg    *config.City
		cfgErr error
	}{
		{name: "nil config", cfg: nil, cfgErr: nil},
		{name: "config load error", cfg: &config.City{Workspace: config.Workspace{Name: "demo"}}, cfgErr: os.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := doctorCheckNames(buildDoctorChecks(cityDir, tt.cfg, tt.cfgErr, buildDoctorChecksOpts{
				ControllerRunning:    false,
				SkipCityDoltCheck:    true,
				SkipManagedDoltCheck: true,
			}))
			if got := doctorCheckIndex(names, "named-always-min-conflict"); got >= 0 {
				t.Fatalf("named-always-min-conflict registered at %d, want absent; names=%v", got, names)
			}
		})
	}
}

func TestBuildDoctorChecksDoltliteUsesPackManagedDoctorChecks(t *testing.T) {
	clearInheritedBeadsEnv(t)

	cityDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(cityDir, "city.toml"), []byte(`[workspace]
name = "demo"

[beads]
provider = "bd"
backend = "doltlite"
`), 0o644); err != nil {
		t.Fatalf("write city.toml: %v", err)
	}

	packDir := filepath.Join(cityDir, ".gc", "system", "packs", "beads-doltlite")
	cfg := &config.City{
		Workspace: config.Workspace{Name: "demo"},
		Beads:     config.BeadsConfig{Provider: "bd", Backend: "doltlite"},
		PackDoctors: []config.DiscoveredDoctor{{
			Name:      "check-sqlite3",
			RunScript: filepath.Join(packDir, "doctor", "check-sqlite3", "run.sh"),
			FixScript: filepath.Join(packDir, "doctor", "check-sqlite3", "fix.sh"),
			PackDir:   packDir,
			PackName:  "beads-doltlite",
		}},
	}

	checks := buildDoctorChecks(cityDir, cfg, nil, buildDoctorChecksOpts{
		ControllerRunning:    false,
		SkipCityDoltCheck:    false,
		SkipManagedDoltCheck: false,
	})
	names := doctorCheckNames(checks)

	for _, legacyName := range []string{"dolt-topology", "dolt-drift", "dolt-server", "dolt-noms-size", "dolt-config", "dolt-version"} {
		if idx := doctorCheckIndex(names, legacyName); idx >= 0 {
			t.Fatalf("%s registered at %d for doltlite city; names=%v", legacyName, idx, names)
		}
	}

	packIdx := doctorCheckIndex(names, "beads-doltlite:check-sqlite3")
	if packIdx < 0 {
		t.Fatalf("pack-managed doltlite doctor check missing; names=%v", names)
	}
	if !checks[packIdx].CanFix() {
		t.Fatalf("pack-managed doltlite doctor check CanFix() = false, want true")
	}
}

func doctorCheckNames(checks []doctor.Check) []string {
	names := make([]string, 0, len(checks))
	for _, check := range checks {
		names = append(names, check.Name())
	}
	return names
}

func doctorCheckIndex(names []string, want string) int {
	for i, name := range names {
		if name == want {
			return i
		}
	}
	return -1
}
