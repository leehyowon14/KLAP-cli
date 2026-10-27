package core

import "testing"

func TestContentBaselineDeduplicationAndReappearance(t *testing.T) {
	state, added, err := DetectContent(SeenContent{}, []ContentItem{{ID: "a"}, {ID: "b"}})
	if err != nil || len(added) != 0 {
		t.Fatalf("baseline: %v %v", added, err)
	}
	next, added, err := DetectContent(state, []ContentItem{{ID: "b", Title: "edited"}, {ID: "c"}, {ID: "c"}})
	if err != nil || len(added) != 1 || added[0].ID != "c" {
		t.Fatalf("new item: %v %v", added, err)
	}
	if state.IDs["c"] {
		t.Fatal("previous state mutated before transaction commit")
	}
	next, added, err = DetectContent(next, []ContentItem{{ID: "c"}, {ID: "a"}, {ID: "b"}})
	if err != nil || len(added) != 0 {
		t.Fatal("reorder/reappearance generated a duplicate")
	}
	unchanged, _, err := DetectContent(next, []ContentItem{{ID: "d"}, {ID: ""}})
	if err == nil || unchanged.IDs["d"] {
		t.Fatal("invalid snapshot advanced the baseline")
	}
}
func TestEmptyInitialSnapshotStillEstablishesBaseline(t *testing.T) {
	s, _, _ := DetectContent(SeenContent{}, nil)
	_, added, err := DetectContent(s, []ContentItem{{ID: "first"}})
	if err != nil || len(added) != 1 {
		t.Fatalf("%v %v", added, err)
	}
}
