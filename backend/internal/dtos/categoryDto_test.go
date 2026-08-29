package dtos

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

func TestToCategoryResponsesNestsChildrenUnderTheirParent(t *testing.T) {
	rows := []database.ListCategoriesRow{
		{Slug: "berries", Name: "Berries"},
		{Slug: "mushrooms", Name: "Mushrooms"},
		{Slug: "chanterelles", Name: "Chanterelles",
			ParentSlug: sql.NullString{String: "mushrooms", Valid: true}},
	}

	got := ToCategoryResponses(rows)

	if len(got) != 2 {
		t.Fatalf("top level = %d entries, want 2 - a child must not appear as one", len(got))
	}

	var mushrooms CategoryResponse
	for _, c := range got {
		if c.Slug == "chanterelles" {
			t.Error("a child was returned at the top level as well as under its parent")
		}
		if c.Slug == "mushrooms" {
			mushrooms = c
		}
	}

	if len(mushrooms.Children) != 1 || mushrooms.Children[0].Slug != "chanterelles" {
		t.Fatalf("mushrooms children = %+v, want one chanterelles", mushrooms.Children)
	}
}

func TestEveryCategoryPublishesAChildrenArray(t *testing.T) {
	rows := []database.ListCategoriesRow{
		{Slug: "mushrooms", Name: "Mushrooms"},
		{Slug: "chanterelles", Name: "Chanterelles",
			ParentSlug: sql.NullString{String: "mushrooms", Valid: true}},
	}

	b, err := json.Marshal(ToCategoryResponses(rows))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	got := string(b)

	want := `[{"slug":"mushrooms","name":"Mushrooms","children":` +
		`[{"slug":"chanterelles","name":"Chanterelles","children":[]}]}]`
	if got != want {
		t.Errorf("json = %s\nwant %s", got, want)
	}
}
