//go:build gascity_doltlite_lib

package beads

/*
#cgo LDFLAGS: -ldoltlite
extern void doltliteInstallAutoExt(void);
static void *gascity_doltlite_link_anchor(void) {
	return (void *)&doltliteInstallAutoExt;
}
*/
import "C"
import "unsafe"

var doltliteLinkAnchor unsafe.Pointer = C.gascity_doltlite_link_anchor()
