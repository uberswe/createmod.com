package search

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"createmod/internal/models"

	"github.com/meilisearch/meilisearch-go"
)

// MeiliIndex is the single Meilisearch index used for search.
const MeiliIndex = "schematics_mods"

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
	Downloads        int64    `json:"downloads"`
	MinecraftVersion string   `json:"minecraft_version"`
	CreateVersion    string   `json:"create_version"`
	CreatedTimestamp  int64    `json:"created_timestamp"`
	BlockCount       int      `json:"block_count"`
	DimX             int      `json:"dim_x"`
	DimY             int      `json:"dim_y"`
	DimZ             int      `json:"dim_z"`
	TrendingScore    float64  `json:"trending_score"`
	RatingCount      int      `json:"rating_count"`
}

// EnsureMeiliIndexes creates the Meilisearch index with proper settings.
func EnsureMeiliIndexes(client meilisearch.ServiceManager) error {
	// Searchable attributes ordered by signal quality (highest first).
	// The "attribute" ranking rule uses this order to boost matches in higher-priority fields.
	searchable := []string{
		"title",
		"ai_description",
		"description",
		"mod_names",
		"author",
		"tags",
		"block_names",
	}

	filterableStr := []string{
		"id", "categories", "minecraft_version", "create_version",
		"tags", "rating", "views", "downloads", "created_timestamp",
		"block_count", "dim_x", "dim_y", "dim_z", "mod_names",
		"trending_score", "rating_count", "modpack_ids",
	}
	filterable := make([]interface{}, len(filterableStr))
	for i, s := range filterableStr {
		filterable[i] = s
	}

	sortable := []string{"rating", "views", "downloads", "created_timestamp", "trending_score", "rating_count"}

	rankingRules := []string{"words", "typo", "proximity", "attribute", "sort", "exactness"}

	synonyms := map[string][]string{
		// Transport
		"train":       {"locomotive", "railway", "rail"},
		"locomotive":  {"train", "railway"},
		"elevator":    {"lift"},
		"lift":        {"elevator"},
		"plane":       {"airplane", "aircraft", "biplane"},
		"airplane":    {"plane", "aircraft"},
		"ship":        {"boat", "vessel", "galleon"},
		"boat":        {"ship", "vessel"},
		"car":         {"automobile", "vehicle"},
		"vehicle":     {"car", "automobile"},
		"helicopter":  {"chopper", "heli"},
		"chopper":     {"helicopter"},

		// Compound word variants (high-volume split/joined searches)
		"airship":    {"air ship", "blimp", "zeppelin", "dirigible"},
		"air ship":   {"airship", "blimp", "zeppelin"},
		"zeppelin":   {"airship", "blimp"},
		"blimp":      {"airship", "zeppelin"},
		"steampunk":  {"steam punk"},
		"steam punk": {"steampunk"},
		"treehouse":  {"tree house"},
		"tree house": {"treehouse"},

		// Buildings & structures
		"house":      {"building", "home"},
		"factory":    {"processing", "production", "refinery"},
		"castle":     {"fortress", "palace", "keep"},
		"fortress":   {"castle", "keep", "stronghold"},
		"chateau":    {"castle", "mansion", "palace"},
		"mansion":    {"manor", "villa", "chateau"},
		"tavern":     {"inn", "pub", "bar"},
		"inn":        {"tavern", "pub"},
		"harbor":     {"harbour", "port", "dock", "marina"},
		"harbour":    {"harbor", "port", "dock", "marina"},
		"port":       {"harbor", "harbour", "dock"},
		"dock":       {"harbor", "harbour", "port", "pier", "wharf"},
		"warehouse":  {"depot", "storehouse"},
		"coliseum":   {"colosseum", "arena", "amphitheater"},
		"colosseum":  {"coliseum", "arena", "amphitheater"},
		"arena":      {"coliseum", "colosseum"},
		"dungeon":    {"prison", "jail", "vault"},

		// Mechanical & Create mod
		"door":        {"gate", "entrance"},
		"gate":        {"door", "entrance"},
		"crane":       {"hoist", "winch"},
		"conveyor":    {"belt"},
		"gear":        {"cog", "cogwheel"},
		"contraption": {"machine", "mechanism", "device"},
		"machine":     {"contraption", "mechanism", "device"},
		"power":       {"energy", "generator"},
		"redstone":    {"logic", "circuitry"},

		// Size
		"compact": {"small", "mini", "tiny"},
		"small":   {"compact", "mini", "tiny"},
		"large":   {"big", "huge", "massive"},
		"big":     {"large", "huge", "massive"},

		// Style & theme
		"decoration": {"decor", "decorative"},
		"medieval":   {"medival", "medievil"},

		// Nature & farming
		"farm":   {"grinder", "harvester"},
		"lumber": {"wood", "timber", "sawmill"},

		// Infrastructure
		"bridge":  {"overpass", "viaduct"},
		"tunnel":  {"underground", "subway"},
		"storage": {"silo", "warehouse"},

		// French
		"maison":      {"house", "home"},
		"usine":       {"factory", "production"},
		"pont":        {"bridge"},
		"gare":        {"station", "train station"},
		"bateau":      {"boat", "ship"},
		"avion":       {"plane", "airplane"},
		"tour":        {"tower"},
		"moulin":      {"mill", "windmill"},
		"ferme":       {"farm"},
		"eglise":      {"church", "cathedral"},
		"phare":       {"lighthouse"},
		"entrepot":    {"warehouse", "storage"},
		"mine":        {"mine", "mineshaft"},

		// German
		"haus":        {"house", "home"},
		"brucke":      {"bridge"},
		"fabrik":      {"factory", "production"},
		"lokomotive":  {"locomotive", "train"},
		"bahnhof":     {"station", "train station"},
		"schiff":      {"ship", "boat"},
		"flugzeug":    {"plane", "airplane"},
		"turm":        {"tower"},
		"burg":        {"castle", "fortress"},
		"schloss":     {"castle", "palace"},
		"muhle":       {"mill", "windmill"},
		"bauernhof":   {"farm"},
		"kirche":      {"church"},
		"leuchtturm":  {"lighthouse"},

		// Spanish
		"tren":        {"train"},
		"casa":        {"house", "home"},
		"puente":      {"bridge"},
		"barco":       {"boat", "ship"},
		"castillo":    {"castle", "fortress"},
		"torre":       {"tower"},
		"iglesia":     {"church", "cathedral"},
		"granja":      {"farm"},
		"molino":      {"mill", "windmill"},
		"estacion":    {"station"},
		"fabrica":     {"factory"},
		"almacen":     {"warehouse", "storage"},
		"faro":        {"lighthouse"},

		// Portuguese
		"ponte":       {"bridge"},
		"navio":       {"ship", "boat"},
		"aviao":       {"plane", "airplane"},
		"castelo":     {"castle"},
		"fazenda":     {"farm"},
		"moinho":      {"mill", "windmill"},
		"estacao":     {"station"},
		"igreja":      {"church"},

		// Dutch
		"trein":       {"train"},
		"brug":        {"bridge"},
		"huis":        {"house", "home"},

		// Russian transliterated
		"poezd":       {"train"},
		"dom":         {"house", "home"},
		"most":        {"bridge"},
		"zamok":       {"castle"},

		// Polish
		"pociag":      {"train"},
		"zamek":       {"castle"},
	}

	stopWords := []string{"the", "a", "an", "is", "it", "of", "for", "with", "and", "or", "in", "on", "to", "my", "this", "that"}

	// Create index if it doesn't exist.
	task, err := client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        MeiliIndex,
		PrimaryKey: "id",
	})
	if err != nil {
		slog.Warn("meili: create index (may already exist)", "uid", MeiliIndex, "error", err)
	} else {
		waitForTask(client, task)
	}

	index := client.Index(MeiliIndex)

	if task, err := index.UpdateSearchableAttributes(&searchable); err != nil {
		slog.Error("meili: update searchable attributes", "error", err)
	} else {
		waitForTask(client, task)
	}

	if task, err := index.UpdateFilterableAttributes(&filterable); err != nil {
		slog.Error("meili: update filterable attributes", "error", err)
	} else {
		waitForTask(client, task)
	}

	if task, err := index.UpdateSortableAttributes(&sortable); err != nil {
		slog.Error("meili: update sortable attributes", "error", err)
	} else {
		waitForTask(client, task)
	}

	if task, err := index.UpdateRankingRules(&rankingRules); err != nil {
		slog.Error("meili: update ranking rules", "error", err)
	} else {
		waitForTask(client, task)
	}

	if task, err := index.UpdateSynonyms(&synonyms); err != nil {
		slog.Error("meili: update synonyms", "error", err)
	} else {
		waitForTask(client, task)
	}

	if task, err := index.UpdateStopWords(&stopWords); err != nil {
		slog.Error("meili: update stop words", "error", err)
	} else {
		waitForTask(client, task)
	}

	if task, err := index.UpdateTypoTolerance(&meilisearch.TypoTolerance{
		Enabled: true,
		MinWordSizeForTypos: meilisearch.MinWordSizeForTypos{
			OneTypo:  4,
			TwoTypos: 7,
		},
	}); err != nil {
		slog.Error("meili: update typo tolerance", "error", err)
	} else {
		waitForTask(client, task)
	}

	if task, err := index.UpdateSeparatorTokens([]string{"_", "-"}); err != nil {
		slog.Error("meili: update separator tokens", "error", err)
	} else {
		waitForTask(client, task)
	}

	if task, err := index.UpdatePagination(&meilisearch.Pagination{
		MaxTotalHits: 5000,
	}); err != nil {
		slog.Error("meili: update pagination", "error", err)
	} else {
		waitForTask(client, task)
	}

	slog.Info("meili: index configured", "uid", MeiliIndex)

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

// SyncMeiliIndex indexes documents into the Meilisearch index.
func SyncMeiliIndex(client meilisearch.ServiceManager, indexUID string, docs []MeiliDocument) error {
	if len(docs) == 0 {
		return nil
	}

	index := client.Index(indexUID)

	// Small batches keep the transient JSON marshal buffers bounded — with
	// large description fields a 1000-doc batch could balloon to hundreds of
	// MB in flight during a rebuild.
	const batchSize = 250
	for start := 0; start < len(docs); start += batchSize {
		end := start + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		batch := docs[start:end]

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
// trendingScores maps schematic IDs to their computed trending scores.
func MapToMeiliDocuments(filterIndex []schematicIndex, trendingScores map[string]float64) []MeiliDocument {
	docs := make([]MeiliDocument, len(filterIndex))
	for i, si := range filterIndex {
		docs[i] = MeiliDocument{
			ID:               si.ID,
			Title:            si.Title,
			Description:      si.Description,
			AIDescription:    si.AIDescription,
			Tags:             si.Tags,
			Categories:       si.Categories,
			Author:           si.Author,
			Rating:           si.Rating,
			Views:            si.Views,
			Downloads:        si.Downloads,
			MinecraftVersion: si.MinecraftVersion,
			CreateVersion:    si.CreateVersion,
			CreatedTimestamp:  si.Created.Unix(),
			BlockNames:       si.BlockNames,
			ModNames:         si.ModNames,
			BlockCount:       si.BlockCount,
			DimX:             si.DimX,
			DimY:             si.DimY,
			DimZ:             si.DimZ,
			RatingCount:      si.RatingCount,
		}
		if trendingScores != nil {
			docs[i].TrendingScore = trendingScores[si.ID]
		}
	}
	return docs
}

// BuildSingleDocument builds a MeiliDocument from a single models.Schematic.
// Used for incremental index updates when a schematic is created or edited.
func BuildSingleDocument(s models.Schematic, modDisplayNames map[string]string, trendingScores map[string]float64) MeiliDocument {
	authorName := ""
	if s.Author != nil {
		authorName = s.Author.Username
	}

	var categories []string
	for _, c := range s.Categories {
		categories = append(categories, c.Name)
	}

	var tags []string
	for _, t := range s.Tags {
		tags = append(tags, t.Name)
	}

	blockNames := ExtractBlockNames(s.Materials)

	var modNames []string
	if modDisplayNames != nil {
		for _, ns := range s.Mods {
			if name, ok := modDisplayNames[ns]; ok && name != "" {
				modNames = append(modNames, name)
			}
		}
	}

	var rating float64
	if parsed, err := strconv.ParseFloat(s.Rating, 64); err == nil {
		rating = parsed
	}

	doc := MeiliDocument{
		ID:               s.ID,
		Title:            stripHtmlRegex(s.Title),
		Description:      stripHtmlRegex(s.Content),
		AIDescription:    stripHtmlRegex(s.AIDescription),
		Tags:             tags,
		Categories:       categories,
		Author:           authorName,
		Rating:           rating,
		Views:            int64(s.Views),
		Downloads:        int64(s.Downloads),
		MinecraftVersion: s.MinecraftVersion,
		CreateVersion:    s.CreatemodVersion,
		CreatedTimestamp:  s.Created.Unix(),
		BlockNames:       blockNames,
		ModNames:         modNames,
		BlockCount:       s.BlockCount,
		DimX:             s.DimX,
		DimY:             s.DimY,
		DimZ:             s.DimZ,
		RatingCount:      s.RatingCount,
	}

	if trendingScores != nil {
		doc.TrendingScore = trendingScores[s.ID]
	}

	return doc
}
