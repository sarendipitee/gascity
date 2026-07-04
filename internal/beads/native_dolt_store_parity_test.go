package beads

import (
	"reflect"
	"sort"
	"testing"
)

func TestNativeDoltStoreWriteReadbackParityWithMemStore(t *testing.T) {
	mem := NewMemStore()
	native := newNativeDoltStoreForTest(newNativeDoltMemStorage())

	memSnap := exerciseStoreWriteReadbackParity(t, mem)
	nativeSnap := exerciseStoreWriteReadbackParity(t, native)

	if !reflect.DeepEqual(nativeSnap, memSnap) {
		t.Fatalf("native DoltLite readback mismatch\nmem:    %#v\nnative: %#v", memSnap, nativeSnap)
	}
}

func exerciseStoreWriteReadbackParity(t *testing.T, store Store) []parityBead {
	t.Helper()

	priority := 1
	parent, err := store.Create(Bead{
		Title:       "parity parent",
		Type:        "molecule",
		Priority:    &priority,
		Description: "parent description",
		Labels:      []string{"alpha", "shared"},
		Metadata:    map[string]string{"phase": "one", "empty": ""},
	})
	if err != nil {
		t.Fatalf("Create parent: %v", err)
	}

	child, err := store.Create(Bead{
		Title:       "parity child",
		Type:        "task",
		ParentID:    parent.ID,
		Priority:    &priority,
		Description: "child description",
		Assignee:    "gascity/worker",
		Labels:      []string{"beta"},
		Metadata:    map[string]string{"child": "yes", "empty": ""},
	})
	if err != nil {
		t.Fatalf("Create child: %v", err)
	}

	inProgress := "in_progress"
	updatedTitle := "parity child updated"
	updatedDescription := "child updated description"
	if err := store.Update(child.ID, UpdateOpts{
		Title:       &updatedTitle,
		Status:      &inProgress,
		Description: &updatedDescription,
		Labels:      []string{"gamma"},
		Metadata:    map[string]string{"child": "updated", "second_empty": ""},
	}); err != nil {
		t.Fatalf("Update child: %v", err)
	}
	if err := store.SetMetadataBatch(parent.ID, map[string]string{"phase": "two", "batch_empty": ""}); err != nil {
		t.Fatalf("SetMetadataBatch parent: %v", err)
	}
	if err := store.Close(parent.ID); err != nil {
		t.Fatalf("Close parent: %v", err)
	}

	beads, err := store.List(ListQuery{AllowScan: true, IncludeClosed: true, TierMode: TierIssues})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return normalizeParityBeads(beads)
}

type parityBead struct {
	Title       string
	Status      string
	Type        string
	Priority    int
	HasPriority bool
	Description string
	Assignee    string
	ParentTitle string
	Labels      []string
	Metadata    map[string]string
}

func normalizeParityBeads(beads []Bead) []parityBead {
	titleByID := make(map[string]string, len(beads))
	for _, b := range beads {
		titleByID[b.ID] = b.Title
	}
	out := make([]parityBead, 0, len(beads))
	for _, b := range beads {
		p := parityBead{
			Title:       b.Title,
			Status:      b.Status,
			Type:        b.Type,
			Description: b.Description,
			Assignee:    b.Assignee,
			ParentTitle: titleByID[b.ParentID],
			Labels:      append([]string(nil), b.Labels...),
			Metadata:    cloneStringMapForParity(b.Metadata),
		}
		if b.Priority != nil {
			p.HasPriority = true
			p.Priority = *b.Priority
		}
		sort.Strings(p.Labels)
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

func cloneStringMapForParity(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
