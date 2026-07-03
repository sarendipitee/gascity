//go:build gascity_doltlite_lib

package main

import (
	"os"
	"strings"

	"github.com/gastownhall/gascity/internal/beads"
)

func openOptimizedDoltliteStore(storePath, cityPath string, store *beads.BdStore) (beads.Store, bool) {
	if !doltliteFastPathScope(cityPath, storePath) {
		return nil, false
	}
	if value := strings.TrimSpace(os.Getenv("GC_BEADS_FORCE_FALLBACK")); value != "" && value != "0" && !strings.EqualFold(value, "false") {
		return store, true
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
