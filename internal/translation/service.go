package translation

import (
	"createmod/internal/cache"
	"createmod/internal/openai"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// SupportedLanguages lists all UI-supported language codes.
var SupportedLanguages = []string{"en", "pt-BR", "pt-PT", "es", "de", "pl", "ru", "zh-Hans"}

// langNames maps ISO codes to human-readable language names for the OpenAI prompt.
var langNames = map[string]string{
	"en":      "English",
	"pt-BR":   "Brazilian Portuguese",
	"pt-PT":   "European Portuguese",
	"es":      "Spanish",
	"de":      "German",
	"pl":      "Polish",
	"ru":      "Russian",
	"zh-Hans": "Simplified Chinese",
}

// Translation holds a single translated record.
type Translation struct {
	ID          string
	Schematic   string
	Language    string
	Title       string
	Description string
	Content     string
}

// GuideTranslation holds a translated guide record.
type GuideTranslation struct {
	ID          string
	Guide       string
	Language    string
	Title       string
	Description string
	Content     string
}

// CollectionTranslation holds a translated collection record.
type CollectionTranslation struct {
	ID          string
	Collection  string
	Language    string
	Title       string
	Description string
}

// Service manages schematic translations via OpenAI.
type Service struct {
	openaiClient *openai.Client
	stopChan     chan struct{}
}

// New creates a new translation service.
func New(apiKey string, logger openai.Logger) *Service {
	return &Service{
		openaiClient: openai.NewClient(apiKey, logger),
		stopChan:     make(chan struct{}),
	}
}

// Stop signals the background scheduler to stop.
func (s *Service) Stop() {
	close(s.stopChan)
}

// StartScheduler starts a background goroutine that backfills missing translations every 30 minutes.
func (s *Service) StartScheduler(app *pocketbase.PocketBase) {
	go func() {
		// Run immediately on start
		s.BackfillMissingTranslations(app)

		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.BackfillMissingTranslations(app)
			case <-s.stopChan:
				app.Logger().Info("Translation scheduler stopped")
				return
			}
		}
	}()
	app.Logger().Info("Translation scheduler started (polling every 30 minutes)")
}

// GetTranslation fetches a translation for a given schematic and language.
// It checks the cache first, then falls back to the database.
func (s *Service) GetTranslation(app *pocketbase.PocketBase, cacheService *cache.Service, schematicID, lang string) *Translation {
	ck := cache.TranslationKey(schematicID, lang)
	if v, found := cacheService.Get(ck); found {
		if t, ok := v.(*Translation); ok {
			return t
		}
	}

	recs, err := app.FindRecordsByFilter(
		"schematic_translations",
		"schematic = {:s} && language = {:l}",
		"-created",
		1,
		0,
		dbx.Params{"s": schematicID, "l": lang},
	)
	if err != nil || len(recs) == 0 {
		return nil
	}

	t := &Translation{
		ID:          recs[0].Id,
		Schematic:   recs[0].GetString("schematic"),
		Language:    recs[0].GetString("language"),
		Title:       recs[0].GetString("title"),
		Description: recs[0].GetString("description"),
		Content:     recs[0].GetString("content"),
	}
	cacheService.SetWithTTL(ck, t, 60*time.Minute)
	return t
}

// SaveOriginalLanguage stores the original text as a translation record for the detected language.
func (s *Service) SaveOriginalLanguage(app *pocketbase.PocketBase, schematicID, lang, title, description, content string) error {
	coll, err := app.FindCollectionByNameOrId("schematic_translations")
	if err != nil {
		return err
	}

	// Check if it already exists
	existing, _ := app.FindRecordsByFilter(
		coll.Id,
		"schematic = {:s} && language = {:l}",
		"-created",
		1,
		0,
		dbx.Params{"s": schematicID, "l": lang},
	)
	if len(existing) > 0 {
		return nil // already saved
	}

	rec := core.NewRecord(coll)
	rec.Set("schematic", schematicID)
	rec.Set("language", lang)
	rec.Set("title", title)
	rec.Set("description", description)
	rec.Set("content", content)
	return app.Save(rec)
}

// TranslateAndSave translates all text fields to a target language in a single API call
// and saves the translation record.
func (s *Service) TranslateAndSave(app *pocketbase.PocketBase, schematicID, targetLang, title, description, content string) error {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return nil
	}

	coll, err := app.FindCollectionByNameOrId("schematic_translations")
	if err != nil {
		return err
	}

	// Check if it already exists
	existing, _ := app.FindRecordsByFilter(
		coll.Id,
		"schematic = {:s} && language = {:l}",
		"-created",
		1,
		0,
		dbx.Params{"s": schematicID, "l": targetLang},
	)
	if len(existing) > 0 {
		return nil // already translated
	}

	langName := langNames[targetLang]
	if langName == "" {
		langName = targetLang
	}

	fields, err := s.openaiClient.TranslateFields(title, description, content, langName)
	if err != nil {
		app.Logger().Debug("translation failed", "lang", targetLang, "error", err)
		return nil
	}

	// Only save if we got at least a title
	if fields.Title == "" {
		return nil
	}

	rec := core.NewRecord(coll)
	rec.Set("schematic", schematicID)
	rec.Set("language", targetLang)
	rec.Set("title", fields.Title)
	rec.Set("description", fields.Description)
	rec.Set("content", fields.Content)
	return app.Save(rec)
}

// TranslateSchematic generates all missing language translations for a single schematic.
func (s *Service) TranslateSchematic(app *pocketbase.PocketBase, schematicID string) {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return
	}

	rec, err := app.FindRecordById("schematics", schematicID)
	if err != nil {
		return
	}

	title := rec.GetString("title")
	description := rec.GetString("description")
	content := rec.GetString("content")
	detectedLang := rec.GetString("detected_language")

	// Save original language if not already saved
	if detectedLang != "" && detectedLang != "en" {
		// The original text might already be overwritten with English at this point,
		// so we check if a record for the detected language already exists
		_ = s.SaveOriginalLanguage(app, schematicID, detectedLang, title, description, content)
	}

	// Use the English version (stored on the main record) as the source for all translations
	for _, lang := range SupportedLanguages {
		if lang == "en" {
			// Save English version from the main record
			_ = s.SaveOriginalLanguage(app, schematicID, "en", title, description, content)
			continue
		}
		err := s.TranslateAndSave(app, schematicID, lang, title, description, content)
		if err != nil {
			app.Logger().Debug("TranslateSchematic: failed to save translation", "id", schematicID, "lang", lang, "error", err)
		}
		// Rate limit: 1 request per second (each TranslateAndSave makes up to 3 API calls)
		time.Sleep(time.Second)
	}
}

// BackfillMissingTranslations finds schematics with fewer than the expected number of
// translation records and generates the missing ones.
func (s *Service) BackfillMissingTranslations(app *pocketbase.PocketBase) {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		app.Logger().Info("BackfillMissingTranslations skipped: OpenAI not configured")
		return
	}
	app.Logger().Info("BackfillMissingTranslations started")

	expectedCount := len(SupportedLanguages)

	// Find schematics that have fewer than expected translations (limit 20 per run)
	// We query approved schematics and then check their translation count
	schematics, err := app.FindRecordsByFilter(
		"schematics",
		"deleted = '' && moderated = 1",
		"-created",
		200,
		0,
	)
	if err != nil {
		app.Logger().Error("BackfillMissingTranslations query failed", "error", err)
		return
	}

	processed := 0
	for _, schematic := range schematics {
		if processed >= 20 {
			break
		}

		// Count existing translations for this schematic
		var count int
		countRecs, err := app.FindRecordsByFilter(
			"schematic_translations",
			"schematic = {:s}",
			"",
			200,
			0,
			dbx.Params{"s": schematic.Id},
		)
		if err == nil {
			count = len(countRecs)
		}

		if count >= expectedCount {
			continue
		}

		app.Logger().Info("BackfillMissingTranslations: translating schematic", "id", schematic.Id, "existing", count, "expected", expectedCount)
		s.TranslateSchematic(app, schematic.Id)
		processed++
		time.Sleep(time.Second)
	}

	app.Logger().Info("BackfillMissingTranslations completed", "processed", processed)

	// Also backfill guides and collections
	s.BackfillGuideTranslations(app)
	s.BackfillCollectionTranslations(app)
}

// ---------- Guide translations ----------

// GetGuideTranslation fetches a translation for a given guide and language.
func (s *Service) GetGuideTranslation(app *pocketbase.PocketBase, cacheService *cache.Service, guideID, lang string) *GuideTranslation {
	ck := cache.GuideTranslationKey(guideID, lang)
	if v, found := cacheService.Get(ck); found {
		if t, ok := v.(*GuideTranslation); ok {
			return t
		}
	}

	recs, err := app.FindRecordsByFilter(
		"guide_translations",
		"guide = {:g} && language = {:l}",
		"-created",
		1,
		0,
		dbx.Params{"g": guideID, "l": lang},
	)
	if err != nil || len(recs) == 0 {
		return nil
	}

	t := &GuideTranslation{
		ID:          recs[0].Id,
		Guide:       recs[0].GetString("guide"),
		Language:    recs[0].GetString("language"),
		Title:       recs[0].GetString("title"),
		Description: recs[0].GetString("description"),
		Content:     recs[0].GetString("content"),
	}
	cacheService.SetWithTTL(ck, t, 60*time.Minute)
	return t
}

// TranslateAndSaveGuide translates guide fields to a target language and saves the record.
func (s *Service) TranslateAndSaveGuide(app *pocketbase.PocketBase, guideID, targetLang, title, description, content string) error {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return nil
	}

	coll, err := app.FindCollectionByNameOrId("guide_translations")
	if err != nil {
		return err
	}

	existing, _ := app.FindRecordsByFilter(
		coll.Id,
		"guide = {:g} && language = {:l}",
		"-created",
		1,
		0,
		dbx.Params{"g": guideID, "l": targetLang},
	)
	if len(existing) > 0 {
		return nil
	}

	langName := langNames[targetLang]
	if langName == "" {
		langName = targetLang
	}

	fields, err := s.openaiClient.TranslateFields(title, description, content, langName)
	if err != nil {
		app.Logger().Debug("guide translation failed", "lang", targetLang, "error", err)
		return nil
	}

	if fields.Title == "" {
		return nil
	}

	rec := core.NewRecord(coll)
	rec.Set("guide", guideID)
	rec.Set("language", targetLang)
	rec.Set("title", fields.Title)
	rec.Set("description", fields.Description)
	rec.Set("content", fields.Content)
	return app.Save(rec)
}

// TranslateGuide generates all missing language translations for a single guide.
func (s *Service) TranslateGuide(app *pocketbase.PocketBase, guideID string) {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return
	}

	rec, err := app.FindRecordById("guides", guideID)
	if err != nil {
		return
	}

	title := rec.GetString("title")
	if title == "" {
		title = rec.GetString("name")
	}
	description := rec.GetString("excerpt")
	content := rec.GetString("content")
	if content == "" {
		content = rec.GetString("content_markdown")
	}
	if content == "" {
		content = rec.GetString("markdown")
	}

	for _, lang := range SupportedLanguages {
		if lang == "en" {
			continue
		}
		err := s.TranslateAndSaveGuide(app, guideID, lang, title, description, content)
		if err != nil {
			app.Logger().Debug("TranslateGuide: failed to save translation", "id", guideID, "lang", lang, "error", err)
		}
		time.Sleep(time.Second)
	}
}

// BackfillGuideTranslations finds guides with fewer than the expected number of
// translation records and generates the missing ones.
func (s *Service) BackfillGuideTranslations(app *pocketbase.PocketBase) {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return
	}
	app.Logger().Info("BackfillGuideTranslations started")

	// Expected translations: all languages minus English (guides are assumed English source)
	expectedCount := len(SupportedLanguages) - 1

	guides, err := app.FindRecordsByFilter(
		"guides",
		"title != ''",
		"-created",
		200,
		0,
	)
	if err != nil {
		app.Logger().Error("BackfillGuideTranslations query failed", "error", err)
		return
	}

	processed := 0
	for _, guide := range guides {
		if processed >= 10 {
			break
		}

		var count int
		countRecs, err := app.FindRecordsByFilter(
			"guide_translations",
			"guide = {:g}",
			"",
			200,
			0,
			dbx.Params{"g": guide.Id},
		)
		if err == nil {
			count = len(countRecs)
		}

		if count >= expectedCount {
			continue
		}

		app.Logger().Info("BackfillGuideTranslations: translating guide", "id", guide.Id, "existing", count, "expected", expectedCount)
		s.TranslateGuide(app, guide.Id)
		processed++
		time.Sleep(time.Second)
	}

	app.Logger().Info("BackfillGuideTranslations completed", "processed", processed)
}

// ---------- Collection translations ----------

// GetCollectionTranslation fetches a translation for a given collection and language.
func (s *Service) GetCollectionTranslation(app *pocketbase.PocketBase, cacheService *cache.Service, collectionID, lang string) *CollectionTranslation {
	ck := cache.CollectionTranslationKey(collectionID, lang)
	if v, found := cacheService.Get(ck); found {
		if t, ok := v.(*CollectionTranslation); ok {
			return t
		}
	}

	recs, err := app.FindRecordsByFilter(
		"collection_translations",
		"collection = {:c} && language = {:l}",
		"-created",
		1,
		0,
		dbx.Params{"c": collectionID, "l": lang},
	)
	if err != nil || len(recs) == 0 {
		return nil
	}

	t := &CollectionTranslation{
		ID:          recs[0].Id,
		Collection:  recs[0].GetString("collection"),
		Language:    recs[0].GetString("language"),
		Title:       recs[0].GetString("title"),
		Description: recs[0].GetString("description"),
	}
	cacheService.SetWithTTL(ck, t, 60*time.Minute)
	return t
}

// TranslateAndSaveCollection translates collection fields to a target language and saves the record.
func (s *Service) TranslateAndSaveCollection(app *pocketbase.PocketBase, collectionID, targetLang, title, description string) error {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return nil
	}

	coll, err := app.FindCollectionByNameOrId("collection_translations")
	if err != nil {
		return err
	}

	existing, _ := app.FindRecordsByFilter(
		coll.Id,
		"collection = {:c} && language = {:l}",
		"-created",
		1,
		0,
		dbx.Params{"c": collectionID, "l": targetLang},
	)
	if len(existing) > 0 {
		return nil
	}

	langName := langNames[targetLang]
	if langName == "" {
		langName = targetLang
	}

	fields, err := s.openaiClient.TranslateFields(title, description, "", langName)
	if err != nil {
		app.Logger().Debug("collection translation failed", "lang", targetLang, "error", err)
		return nil
	}

	if fields.Title == "" {
		return nil
	}

	rec := core.NewRecord(coll)
	rec.Set("collection", collectionID)
	rec.Set("language", targetLang)
	rec.Set("title", fields.Title)
	rec.Set("description", fields.Description)
	return app.Save(rec)
}

// TranslateCollection generates all missing language translations for a single collection.
func (s *Service) TranslateCollection(app *pocketbase.PocketBase, collectionID string) {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return
	}

	rec, err := app.FindRecordById("collections", collectionID)
	if err != nil {
		return
	}

	title := rec.GetString("title")
	if title == "" {
		title = rec.GetString("name")
	}
	description := rec.GetString("description")

	for _, lang := range SupportedLanguages {
		if lang == "en" {
			continue
		}
		err := s.TranslateAndSaveCollection(app, collectionID, lang, title, description)
		if err != nil {
			app.Logger().Debug("TranslateCollection: failed to save translation", "id", collectionID, "lang", lang, "error", err)
		}
		time.Sleep(time.Second)
	}
}

// BackfillCollectionTranslations finds published collections with fewer than the expected
// number of translation records and generates the missing ones.
func (s *Service) BackfillCollectionTranslations(app *pocketbase.PocketBase) {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return
	}
	app.Logger().Info("BackfillCollectionTranslations started")

	expectedCount := len(SupportedLanguages) - 1

	collections, err := app.FindRecordsByFilter(
		"collections",
		"published = true && title != ''",
		"-created",
		200,
		0,
	)
	if err != nil {
		app.Logger().Error("BackfillCollectionTranslations query failed", "error", err)
		return
	}

	processed := 0
	for _, coll := range collections {
		if processed >= 10 {
			break
		}

		var count int
		countRecs, err := app.FindRecordsByFilter(
			"collection_translations",
			"collection = {:c}",
			"",
			200,
			0,
			dbx.Params{"c": coll.Id},
		)
		if err == nil {
			count = len(countRecs)
		}

		if count >= expectedCount {
			continue
		}

		app.Logger().Info("BackfillCollectionTranslations: translating collection", "id", coll.Id, "existing", count, "expected", expectedCount)
		s.TranslateCollection(app, coll.Id)
		processed++
		time.Sleep(time.Second)
	}

	app.Logger().Info("BackfillCollectionTranslations completed", "processed", processed)
}
