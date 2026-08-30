package dtos

import (
	"github.com/IbnBaqqi/transcendence/internal/database"
)

type CategoryResponse struct {
	Slug     string             `json:"slug"`
	Name     string             `json:"name"`
	Children []CategoryResponse `json:"children"`
}

func ToCategoryResponses(rows []database.ListCategoriesRow) []CategoryResponse {
	out := make([]CategoryResponse, 0, len(rows))
	at := make(map[string]int, len(rows))

	for _, row := range rows {
		if row.ParentSlug.Valid {
			continue
		}
		at[row.Slug] = len(out)
		out = append(out, CategoryResponse{
			Slug:     row.Slug,
			Name:     row.Name,
			Children: []CategoryResponse{},
		})
	}

	for _, row := range rows {
		if !row.ParentSlug.Valid {
			continue
		}
		parent, ok := at[row.ParentSlug.String]
		if !ok {
			continue
		}
		out[parent].Children = append(out[parent].Children, CategoryResponse{
			Slug:     row.Slug,
			Name:     row.Name,
			Children: []CategoryResponse{},
		})
	}

	return out
}
