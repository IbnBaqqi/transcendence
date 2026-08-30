-- name: ListCategories :many
SELECT slug, name, parent_slug FROM categories
ORDER BY COALESCE(parent_slug, slug), parent_slug NULLS FIRST, slug;
