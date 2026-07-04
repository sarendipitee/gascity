//go:build !gascity_doltlite_lib

package main

import "github.com/gastownhall/gascity/internal/beads"

func openOptimizedDoltliteStore(storePath, cityPath string, store *beads.BdStore) (beads.Store, bool) {
	if !scopeBackendIsDoltlite(cityPath, resolveStoreScopeRoot(cityPath, storePath)) {
		return nil, false
	}
	plugin, err := beads.NewBackendPluginStore(storePath, store)
	if err == nil {
		return plugin, true
	}
	return nil, false
}
