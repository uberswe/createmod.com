package search

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

// Meilisearch index UIDs for each enrichment level.
const (
	MeiliIndexBase = "schematics_base"
	MeiliIndexAI   = "schematics_ai"
	MeiliIndexFull = "schematics_full"
	MeiliIndexMods = "schematics_mods"
)

// MeiliDocument represents a schematic document in Meilisearch.
type MeiliDocument struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	AIDescription    string   `json:"ai_description,omitempty"`
	Tags             []string `json:"tags"`
	Categories       []string `json:"categories"`
	Author           string   `json:"author"`
	BlockNames       []string `json:"block_names,omitempty"`
	ModNames         []string `json:"mod_names,omitempty"`
	Rating           float64  `json:"rating"`
	Views            int64    `json:"views"`
	Paid             bool     `json:"paid"`
	MinecraftVersion string   `json:"minecraft_version"`
	CreateVersion    string   `json:"create_version"`
	CreatedTimestamp  int64    `json:"created_timestamp"`
}

// EnsureMeiliIndexes creates the three Meilisearch indexes with proper settings.
func EnsureMeiliIndexes(client meilisearch.ServiceManager) error {
	indexes := []struct {
		UID        string
		Searchable []string
	}{
		{
			UID:        MeiliIndexBase,
			Searchable: []string{"title", "tags", "description", "author"},
		},
		{
			UID:        MeiliIndexAI,
			Searchable: []string{"title", "tags", "description", "ai_description", "author"},
		},
		{
			UID:        MeiliIndexFull,
			Searchable: []string{"title", "tags", "block_names", "description", "ai_description", "author"},
		},
		{
			UID:        MeiliIndexMods,
			Searchable: []string{"title", "tags", "block_names", "mod_names", "description", "ai_description", "author"},
		},
	}

	filterableStr := []string{
		"categories", "minecraft_version", "create_version",
		"tags", "rating", "paid", "views", "created_timestamp",
	}
	filterable := make([]interface{}, len(filterableStr))
	for i, s := range filterableStr {
		filterable[i] = s
	}
	sortable := []string{"rating", "views", "created_timestamp"}

	for _, idx := range indexes {
		// Create index if it doesn't exist.
		task, err := client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        idx.UID,
			PrimaryKey: "id",
		})
		if err != nil {
			slog.Warn("meili: create index (may already exist)", "uid", idx.UID, "error", err)
		} else {
			waitForTask(client, task)
		}

		index := client.Index(idx.UID)

		// Set searchable attributes.
		if task, err := index.UpdateSearchableAttributes(&idx.Searchable); err != nil {
			slog.Error("meili: update searchable attributes", "uid", idx.UID, "error", err)
		} else {
			waitForTask(client, task)
		}

		// Set filterable attributes.
		if task, err := index.UpdateFilterableAttributes(&filterable); err != nil {
			slog.Error("meili: update filterable attributes", "uid", idx.UID, "error", err)
		} else {
			waitForTask(client, task)
		}

		// Set sortable attributes.
		if task, err := index.UpdateSortableAttributes(&sortable); err != nil {
			slog.Error("meili: update sortable attributes", "uid", idx.UID, "error", err)
		} else {
			waitForTask(client, task)
		}

		slog.Info("meili: index configured", "uid", idx.UID)
	}

	return nil
}

// waitForTask blocks until a Meilisearch task completes or times out.
func waitForTask(client meilisearch.ServiceManager, taskInfo *meilisearch.TaskInfo) {
	if taskInfo == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Warn("meili: timed out waiting for task", "taskUID", taskInfo.TaskUID)
			return
		case <-ticker.C:
			task, err := client.GetTask(taskInfo.TaskUID)
			if err != nil {
				slog.Warn("meili: error checking task", "error", err)
				return
			}
			if task.Status == meilisearch.TaskStatusSucceeded || task.Status == meilisearch.TaskStatusFailed {
				if task.Status == meilisearch.TaskStatusFailed {
					slog.Warn("meili: task failed", "taskUID", taskInfo.TaskUID, "error", task.Error)
				}
				return
			}
		}
	}
}

// SyncMeiliIndex indexes documents into a specific Meilisearch index.
// It strips fields not relevant to the index level.
func SyncMeiliIndex(client meilisearch.ServiceManager, indexUID string, docs []MeiliDocument) error {
	if len(docs) == 0 {
		return nil
	}

	// Strip fields based on index level.
	cleaned := make([]MeiliDocument, len(docs))
	copy(cleaned, docs)
	switch indexUID {
	case MeiliIndexBase:
		for i := range cleaned {
			cleaned[i].AIDescription = ""
			cleaned[i].BlockNames = nil
			cleaned[i].ModNames = nil
		}
	case MeiliIndexAI:
		for i := range cleaned {
			cleaned[i].BlockNames = nil
			cleaned[i].ModNames = nil
		}
	case MeiliIndexFull:
		for i := range cleaned {
			cleaned[i].ModNames = nil
		}
	case MeiliIndexMods:
		// Keep everything including ModNames.
	}

	index := client.Index(indexUID)

	// Batch in groups of 1000.
	const batchSize = 1000
	for start := 0; start < len(cleaned); start += batchSize {
		end := start + batchSize
		if end > len(cleaned) {
			end = len(cleaned)
		}
		batch := cleaned[start:end]

		pk := "id"
		task, err := index.AddDocuments(batch, &meilisearch.DocumentOptions{PrimaryKey: &pk})
		if err != nil {
			return fmt.Errorf("meili: add documents to %s (batch %d-%d): %w", indexUID, start, end, err)
		}
		waitForTask(client, task)
	}

	return nil
}

// MapToMeiliDocuments converts schematic index entries to Meilisearch documents.
func MapToMeiliDocuments(filterIndex []schematicIndex, bleveEntries []indexCacheEntry) []MeiliDocument {
	docs := make([]MeiliDocument, len(filterIndex))
	for i, si := range filterIndex {
		doc := MeiliDocument{
			ID:               si.ID,
			Title:            si.Title,
			Description:      si.Description,
			Tags:             si.Tags,
			Categories:       si.Categories,
			Author:           si.Author,
			Rating:           si.Rating,
			Views:            si.Views,
			Paid:             si.Paid,
			MinecraftVersion: si.MinecraftVersion,
			CreateVersion:    si.CreateVersion,
			CreatedTimestamp:  si.Created.Unix(),
			BlockNames:       si.BlockNames,
			ModNames:         si.ModNames,
		}
		// Pull AIDescription from the Bleve cache entry if available.
		if i < len(bleveEntries) {
			doc.AIDescription = bleveEntries[i].BI.AIDescription
		}
		docs[i] = doc
	}
	return docs
}
