//go:build !gascity_doltlite_lib

package beads

// doltliteSQLDriverName names the pure-Go sqlite driver used by non-libdoltlite
// builds and tests.
const doltliteSQLDriverName = "sqlite"
