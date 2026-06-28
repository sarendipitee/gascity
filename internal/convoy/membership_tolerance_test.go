package convoy

import (
	"fmt"
	"testing"

	"github.com/gastownhall/gascity/internal/beads"
)

// getErrorStore wraps a Store and makes Get fail with a non-NotFound error for
// one bead ID, simulating a tracked bead whose row exists but cannot be
// materialized (e.g. corrupt metadata JSON the SQL/JSON layer can't parse).
type getErrorStore struct {
	beads.Store
	badID string
}

func (s getErrorStore) Get(id string) (beads.Bead, error) {
	if id == s.badID {
		return beads.Bead{}, fmt.Errorf("getting issue %s: unmarshaling metadata: invalid character 'm' looking for beginning of value", id)
	}
	return s.Store.Get(id)
}

// TestConvoyMembersToleratesUnreadableTrackedBead reproduces the gc-7ix DoS:
// one tracked bead that fails to load (not ErrNotFound) must not abort the whole
// convoy pass. It should degrade to an unresolved (unknown-status) tracked item
// alongside the healthy members.
func TestConvoyMembersToleratesUnreadableTrackedBead(t *testing.T) {
	mem := beads.NewMemStore()

	convoy, _ := mem.Create(beads.Bead{Title: "convoy", Type: "convoy"})
	good, _ := mem.Create(beads.Bead{Title: "good task"})
	bad, _ := mem.Create(beads.Bead{Title: "corrupt task"})
	if err := mem.DepAdd(convoy.ID, good.ID, "tracks"); err != nil {
		t.Fatal(err)
	}
	if err := mem.DepAdd(convoy.ID, bad.ID, "tracks"); err != nil {
		t.Fatal(err)
	}

	store := getErrorStore{Store: mem, badID: bad.ID}

	members, err := Members(store, convoy.ID, true)
	if err != nil {
		t.Fatalf("Members aborted on one unreadable bead: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("members = %d, want 2 (healthy + degraded)", len(members))
	}

	byID := make(map[string]beads.Bead, len(members))
	for _, m := range members {
		byID[m.ID] = m
	}
	if _, ok := byID[good.ID]; !ok {
		t.Errorf("healthy member %s missing from results", good.ID)
	}
	degraded, ok := byID[bad.ID]
	if !ok {
		t.Fatalf("unreadable member %s missing from results", bad.ID)
	}
	if degraded.Status != "unknown" {
		t.Errorf("unreadable member status = %q, want unknown (non-terminal)", degraded.Status)
	}
	if !IsUnresolvedTrackedItem(degraded) {
		t.Errorf("unreadable member should be reported as an unresolved tracked item")
	}
}
