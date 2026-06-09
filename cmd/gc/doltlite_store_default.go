//go:build !gascity_native_beads && !gascity_doltlite_lib

package main

import "github.com/gastownhall/gascity/internal/beads"

func openOptimizedDoltliteStore(_, _ string, _ *beads.BdStore) (beads.Store, bool) {
	return nil, false
}
