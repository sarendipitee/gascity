package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gastownhall/gascity/internal/beads/contract"
	"github.com/gastownhall/gascity/internal/fsys"
)

// BeadsBackend abstracts bead storage backend behavior so callers
// dispatch on backend identity in one place instead of branching on
// cityUsesDoltliteBeadsBackend() across the codebase.
type BeadsBackend interface {
	Name() string
	NeedsManagedServer() bool
	NeedsDoltBinary() bool
	MinBDVersion() string
	NeedsBeadHooks() bool
	NeedsDoltDoctorChecks() bool
	MetadataInit(fs fsys.FS, scopeRoot, doltDatabase string, preserveExisting bool) error
	MetadataEnforce(fs fsys.FS, scopeRoot, doltDatabase string) error
	ProviderEnv() []string
	RequiredBuiltinPacks() []string
}

type doltBackend struct{}

func (d *doltBackend) Name() string                   { return "dolt" }
func (d *doltBackend) NeedsManagedServer() bool       { return true }
func (d *doltBackend) NeedsDoltBinary() bool          { return true }
func (d *doltBackend) MinBDVersion() string           { return "1.0.4" }
func (d *doltBackend) NeedsBeadHooks() bool           { return true }
func (d *doltBackend) NeedsDoltDoctorChecks() bool    { return true }
func (d *doltBackend) RequiredBuiltinPacks() []string { return []string{"dolt"} }

func (d *doltBackend) MetadataInit(fs fsys.FS, scopeRoot, doltDatabase string, preserveExisting bool) error {
	return ensureCanonicalScopeMetadata(fs, scopeRoot, doltDatabase, preserveExisting)
}

func (d *doltBackend) MetadataEnforce(fs fsys.FS, scopeRoot, doltDatabase string) error {
	return enforceCanonicalScopeMetadataForInit(fs, scopeRoot, doltDatabase)
}

func (d *doltBackend) ProviderEnv() []string { return nil }

type doltliteBackend struct{}

func (dl *doltliteBackend) Name() string                   { return "doltlite" }
func (dl *doltliteBackend) NeedsManagedServer() bool       { return false }
func (dl *doltliteBackend) NeedsDoltBinary() bool          { return false }
func (dl *doltliteBackend) MinBDVersion() string           { return "1.0.3" }
func (dl *doltliteBackend) NeedsBeadHooks() bool           { return false }
func (dl *doltliteBackend) NeedsDoltDoctorChecks() bool    { return false }
func (dl *doltliteBackend) RequiredBuiltinPacks() []string { return []string{"beads-doltlite-init"} }

func (dl *doltliteBackend) MetadataInit(fs fsys.FS, scopeRoot, doltDatabase string, preserveExisting bool) error {
	return ensureCanonicalDoltliteScopeMetadata(fs, scopeRoot, doltDatabase, preserveExisting)
}

func (dl *doltliteBackend) MetadataEnforce(fs fsys.FS, scopeRoot, doltDatabase string) error {
	return enforceCanonicalDoltliteScopeMetadataForInit(fs, scopeRoot, doltDatabase)
}

func (dl *doltliteBackend) ProviderEnv() []string {
	return []string{"GC_BEADS_BACKEND=doltlite", "BEADS_BACKEND=doltlite"}
}

type externalBeadsBackend struct {
	name string
}

func (b *externalBeadsBackend) Name() string                { return b.name }
func (b *externalBeadsBackend) NeedsManagedServer() bool    { return false }
func (b *externalBeadsBackend) NeedsDoltBinary() bool       { return false }
func (b *externalBeadsBackend) MinBDVersion() string        { return "1.0.3" }
func (b *externalBeadsBackend) NeedsBeadHooks() bool        { return false }
func (b *externalBeadsBackend) NeedsDoltDoctorChecks() bool { return false }
func (b *externalBeadsBackend) ProviderEnv() []string       { return nil }
func (b *externalBeadsBackend) RequiredBuiltinPacks() []string {
	return nil
}

func (b *externalBeadsBackend) MetadataInit(_ fsys.FS, _ string, _ string, _ bool) error {
	return fmt.Errorf("beads backend %q does not support managed metadata initialization", b.name)
}

func (b *externalBeadsBackend) MetadataEnforce(_ fsys.FS, _ string, _ string) error {
	return fmt.Errorf("beads backend %q does not support managed metadata enforcement", b.name)
}

// resolveBeadsBackend returns the active backend for a city path.
func resolveBeadsBackend(cityPath string) BeadsBackend {
	return resolveBeadsBackendName(resolveBeadsBackendString(cityPath))
}

func resolveBeadsBackendName(name string) BeadsBackend {
	backend := strings.ToLower(strings.TrimSpace(name))
	if backend == "doltlite" {
		return &doltliteBackend{}
	}
	if backend == "postgres" {
		return &externalBeadsBackend{name: "postgres"}
	}
	return &doltBackend{}
}

func resolveBeadsBackendString(cityPath string) string {
	if v := strings.TrimSpace(os.Getenv("GC_BEADS_BACKEND")); v != "" {
		return v
	}
	return strings.TrimSpace(peekBeadsBackend(filepath.Join(cityPath, "city.toml")))
}

func resolveScopeBeadsBackend(cityPath, scopeRoot string) BeadsBackend {
	scopeRoot = strings.TrimSpace(scopeRoot)
	if scopeRoot == "" {
		return resolveBeadsBackend(cityPath)
	}
	if !filepath.IsAbs(scopeRoot) {
		scopeRoot = filepath.Join(cityPath, scopeRoot)
	}

	cityBackend := resolveBeadsBackend(cityPath)
	if samePath(cityPath, scopeRoot) {
		return cityBackend
	}

	resolved, err := contract.ResolveScopeConfigState(fsys.OSFS{}, cityPath, scopeRoot, "")
	if err == nil &&
		resolved.Kind == contract.ScopeConfigAuthoritative &&
		resolved.State.EndpointOrigin == contract.EndpointOriginInheritedCity {
		return cityBackend
	}

	meta, ok, err := contract.LoadMetadataState(fsys.OSFS{}, scopeMetadataJSONPath(scopeRoot))
	if err == nil && ok && strings.TrimSpace(meta.Backend) != "" {
		return resolveBeadsBackendName(meta.Backend)
	}
	if !scopeOverridesCityBackend(cityPath, scopeRoot) {
		return cityBackend
	}
	return &doltBackend{}
}

func scopeNeedsDoltDoctorChecks(cityPath, scopeRoot string) bool {
	return resolveScopeBeadsBackend(cityPath, scopeRoot).NeedsDoltDoctorChecks()
}
