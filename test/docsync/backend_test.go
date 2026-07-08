package docsync

import (
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("GC_BEADS_BACKEND")), "doltlite") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("BEADS_BACKEND")), "doltlite") {
		os.Exit(0)
	}
	os.Exit(m.Run())
}
