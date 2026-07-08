//go:build gascity_doltlite_lib

package main

import (
	"github.com/gastownhall/gascity/internal/beads"
)

func openOptimizedDoltliteStore(storePath, cityPath string, store *beads.BdStore) (beads.Store, bool) {
	if !doltliteFastPathScope(cityPath, storePath) {
		return nil, false
	}
	plugin, err := beads.NewBackendPluginStore(storePath, store)
	if err == nil {
		return plugin, true
	}
	direct, err := beads.NewDoltliteReadStore(storePath, store)
	if err == nil {
		return direct, true
	}
	return nil, false
}

func doltliteFastPathScope(cityPath, storePath string) bool {
	return scopeBackendIsDoltlite(cityPath, resolveStoreScopeRoot(cityPath, storePath))
}
