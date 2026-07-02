package main

import "github.com/gastownhall/gascity/internal/config"

// doltliteLoaderEnvScrub returns env assignments that force child launchers to
// remove ambient dynamic-loader paths. DoltLite-linked bd/gc binaries must use
// the libdoltlite selected by the pack installer, not a shell or tmux server
// value left over from a local development tree.
func doltliteLoaderEnvScrub() map[string]string {
	return map[string]string{
		"LD_LIBRARY_PATH":   "",
		"DYLD_LIBRARY_PATH": "",
	}
}

func cityUsesDoltliteBackend(cfg *config.City) bool {
	if cfg == nil {
		return false
	}
	return resolveBeadsBackendName(cfg.Beads.Backend).Name() == "doltlite"
}
