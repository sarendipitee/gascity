package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gastownhall/gascity/internal/beads"
	"github.com/gastownhall/gascity/internal/beads/contract"
	"github.com/gastownhall/gascity/internal/fsys"
)

// BeadsBackendCapabilities advertises optional backend behavior that Gas City
// may use without hard-coding the concrete backend name at each call site.
type BeadsBackendCapabilities struct {
	ManagedServer       bool
	DoltBinary          bool
	BeadHooks           bool
	DoltDoctorChecks    bool
	OptimizedLocalStore bool
	OptimizedStoreName  string
}

// BeadsBackend abstracts bead storage backend behavior so callers
// dispatch on backend identity in one place instead of branching on
// cityUsesDoltliteBeadsBackend() across the codebase.
type BeadsBackend interface {
	Name() string
	Capabilities() BeadsBackendCapabilities
	NeedsManagedServer() bool
	NeedsDoltBinary() bool
	MinBDVersion() string
	NeedsBeadHooks() bool
	NeedsDoltDoctorChecks() bool
	MetadataInit(fs fsys.FS, scopeRoot, doltDatabase string, preserveExisting bool) error
	MetadataEnforce(fs fsys.FS, scopeRoot, doltDatabase string) error
	ProviderEnv() []string
	RequiredBuiltinPacks() []string
	OpenOptimizedStore(scopeRoot, cityPath string, store *beads.BdStore) (beads.Store, bool)
}

type doltBackend struct{}

func (d *doltBackend) Name() string { return "dolt" }
func (d *doltBackend) Capabilities() BeadsBackendCapabilities {
	return BeadsBackendCapabilities{
		ManagedServer:    true,
		DoltBinary:       true,
		BeadHooks:        true,
		DoltDoctorChecks: true,
	}
}
func (d *doltBackend) NeedsManagedServer() bool       { return d.Capabilities().ManagedServer }
func (d *doltBackend) NeedsDoltBinary() bool          { return d.Capabilities().DoltBinary }
func (d *doltBackend) MinBDVersion() string           { return "1.0.4" }
func (d *doltBackend) NeedsBeadHooks() bool           { return d.Capabilities().BeadHooks }
func (d *doltBackend) NeedsDoltDoctorChecks() bool    { return d.Capabilities().DoltDoctorChecks }
func (d *doltBackend) RequiredBuiltinPacks() []string { return []string{"dolt"} }

func (d *doltBackend) MetadataInit(fs fsys.FS, scopeRoot, doltDatabase string, preserveExisting bool) error {
	return ensureCanonicalScopeMetadata(fs, scopeRoot, doltDatabase, preserveExisting)
}

func (d *doltBackend) MetadataEnforce(fs fsys.FS, scopeRoot, doltDatabase string) error {
	return enforceCanonicalScopeMetadataForInit(fs, scopeRoot, doltDatabase)
}

func (d *doltBackend) ProviderEnv() []string { return nil }

func (d *doltBackend) OpenOptimizedStore(_, _ string, _ *beads.BdStore) (beads.Store, bool) {
	return nil, false
}

type doltliteBackend struct{}

func (dl *doltliteBackend) Name() string { return "doltlite" }
func (dl *doltliteBackend) Capabilities() BeadsBackendCapabilities {
	return BeadsBackendCapabilities{
		OptimizedLocalStore: true,
		OptimizedStoreName:  beads.BeadsStoreNameBackendPluginStore,
	}
}
func (dl *doltliteBackend) NeedsManagedServer() bool       { return dl.Capabilities().ManagedServer }
func (dl *doltliteBackend) NeedsDoltBinary() bool          { return dl.Capabilities().DoltBinary }
func (dl *doltliteBackend) MinBDVersion() string           { return "1.0.3" }
func (dl *doltliteBackend) NeedsBeadHooks() bool           { return dl.Capabilities().BeadHooks }
func (dl *doltliteBackend) NeedsDoltDoctorChecks() bool    { return dl.Capabilities().DoltDoctorChecks }
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

func (dl *doltliteBackend) OpenOptimizedStore(scopeRoot, cityPath string, store *beads.BdStore) (beads.Store, bool) {
	return openOptimizedDoltliteStore(scopeRoot, cityPath, store)
}

type externalBeadsBackend struct {
	name string
}

func (b *externalBeadsBackend) Name() string { return b.name }
func (b *externalBeadsBackend) Capabilities() BeadsBackendCapabilities {
	return BeadsBackendCapabilities{}
}
func (b *externalBeadsBackend) NeedsManagedServer() bool    { return b.Capabilities().ManagedServer }
func (b *externalBeadsBackend) NeedsDoltBinary() bool       { return b.Capabilities().DoltBinary }
func (b *externalBeadsBackend) MinBDVersion() string        { return "1.0.3" }
func (b *externalBeadsBackend) NeedsBeadHooks() bool        { return b.Capabilities().BeadHooks }
func (b *externalBeadsBackend) NeedsDoltDoctorChecks() bool { return b.Capabilities().DoltDoctorChecks }
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

func (b *externalBeadsBackend) OpenOptimizedStore(_, _ string, _ *beads.BdStore) (beads.Store, bool) {
	return nil, false
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
