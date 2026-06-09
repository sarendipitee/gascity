//go:build gascity_native_beads || gascity_doltlite_lib

package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/gastownhall/gascity/internal/beads"
)

const nativeDoltliteBeadsEnv = "GC_NATIVE_DOLTLITE_BEADS"

func openOptimizedDoltliteStore(storePath, cityPath string, store *beads.BdStore) (beads.Store, bool) {
	if !nativeDoltliteBeadsEnabled(cityPath, storePath) {
		return nil, false
	}
	direct, err := beads.NewDoltliteReadStore(storePath, store)
	if err == nil {
		return direct, true
	}
	return nil, false
}

func nativeDoltliteBeadsEnabled(cityPath, storePath string) bool {
	raw := strings.TrimSpace(os.Getenv(nativeDoltliteBeadsEnv))
	if raw != "" {
		enabled, err := strconv.ParseBool(raw)
		return err == nil && enabled
	}
	return scopeBackendIsDoltlite(cityPath, resolveStoreScopeRoot(cityPath, storePath))
}
