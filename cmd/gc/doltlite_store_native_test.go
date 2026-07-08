//go:build gascity_doltlite_lib

package main

import "testing"

func TestDoltliteReadFastPathEnabled(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		want    bool
	}{
		{name: "doltlite backend", backend: "doltlite", want: true},
		{name: "dolt backend", backend: "dolt", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cityPath := t.TempDir()
			t.Setenv("GC_BEADS_BACKEND", tt.backend)
			if got := doltliteFastPathScope(cityPath, cityPath); got != tt.want {
				t.Fatalf("doltliteFastPathScope() = %v, want %v", got, tt.want)
			}
		})
	}
}
