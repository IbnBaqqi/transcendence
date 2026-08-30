package dtos

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

var (
	tagged   = uuid.MustParse("11111111-1111-4111-8111-111111111111")
	untagged = uuid.MustParse("22222222-2222-4222-8222-222222222222")
)

func TestWithTagsPutsTheTagsOnTheResponse(t *testing.T) {
	got := WithTags(ListingResponse{ID: tagged, Tags: []string{}}, []string{"chanterelle", "roadside"})

	if len(got.Tags) != 2 || got.Tags[0] != "chanterelle" || got.Tags[1] != "roadside" {
		t.Fatalf("tags = %q, want [chanterelle roadside]", got.Tags)
	}
}

func TestWithTagsLeavesAnEmptyArrayWhenThereAreNone(t *testing.T) {
	got := WithTags(ListingResponse{ID: tagged, Tags: []string{}}, nil)

	if got.Tags == nil {
		t.Fatal("tags became nil, which marshals to null")
	}
	if len(got.Tags) != 0 {
		t.Errorf("tags = %q, want empty", got.Tags)
	}

	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(b), `"tags":[]`) {
		t.Errorf("json = %s, want tags as an empty array", b)
	}
}

func TestWithTagsEachMatchesTagsToTheirOwnListing(t *testing.T) {
	items := []ListingResponse{
		{ID: tagged, Tags: []string{}},
		{ID: untagged, Tags: []string{}},
	}

	got := WithTagsEach(items, map[uuid.UUID][]string{tagged: {"chanterelle"}})

	if len(got[0].Tags) != 1 || got[0].Tags[0] != "chanterelle" {
		t.Errorf("the tagged listing got %q, want [chanterelle]", got[0].Tags)
	}
	if len(got[1].Tags) != 0 {
		t.Errorf("the untagged listing got %q, want none", got[1].Tags)
	}
}
