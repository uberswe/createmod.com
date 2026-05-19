package pages

import (
	"context"
	"createmod/internal/store"
	"log/slog"
	"strings"

	"github.com/gosimple/slug"
)

const maxTagNameLen = 50

// resolveTagIDs takes a list of form values that may be existing tag IDs or new tag names.
// For new names, it creates a pending tag (public=false) in the database.
// Returns a list of tag IDs (existing + newly created).
func resolveTagIDs(ctx context.Context, appStore *store.Store, values []string) []string {
	var ids []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}

		// Try as an existing tag ID first (covers any ID format).
		if _, err := appStore.Tags.GetByID(ctx, v); err == nil {
			ids = append(ids, v)
			continue
		}

		// Reject names that are too long.
		if len(v) > maxTagNameLen {
			continue
		}

		// Treat as a new tag name suggestion.
		key := slug.Make(v)
		if key == "" {
			continue
		}

		// Check if a tag with this key already exists (including pending).
		existing, err := appStore.Tags.GetByKey(ctx, key)
		if err == nil && existing != nil {
			ids = append(ids, existing.ID)
			continue
		}

		// Create a new pending tag.
		t := &store.Tag{
			Key:    key,
			Name:   v,
			Public: false,
		}
		if err := appStore.Tags.Create(ctx, t); err != nil {
			slog.Warn("tag_suggest: failed to create pending tag", "name", v, "error", err)
			continue
		}
		ids = append(ids, t.ID)
	}
	return ids
}

// resolveCategoryIDs takes a list of form values that may be existing category IDs or new category names.
// For new names, it creates a pending category (public=false) in the database.
// Returns a list of category IDs (existing + newly created).
func resolveCategoryIDs(ctx context.Context, appStore *store.Store, values []string) []string {
	var ids []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}

		// Try as an existing category ID first (covers any ID format).
		if _, err := appStore.Categories.GetByID(ctx, v); err == nil {
			ids = append(ids, v)
			continue
		}

		// Reject names that are too long.
		if len(v) > maxTagNameLen {
			continue
		}

		// Treat as a new category name suggestion.
		key := slug.Make(v)
		if key == "" {
			continue
		}

		// Check if a category with this key already exists (including pending).
		existing, err := appStore.Categories.GetByKey(ctx, key)
		if err == nil && existing != nil {
			ids = append(ids, existing.ID)
			continue
		}

		// Create a new pending category.
		c := &store.Category{
			Key:    key,
			Name:   v,
			Public: false,
		}
		if err := appStore.Categories.Create(ctx, c); err != nil {
			slog.Warn("tag_suggest: failed to create pending category", "name", v, "error", err)
			continue
		}
		ids = append(ids, c.ID)
	}
	return ids
}
