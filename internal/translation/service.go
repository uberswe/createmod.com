package translation

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/openai"
	"createmod/internal/store"
	"log/slog"
	"time"
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
	appStore     *store.Store
}

// New creates a new translation service.
func New(apiKey string, logger openai.Logger, appStore *store.Store) *Service {
	return &Service{
		openaiClient: openai.NewClient(apiKey, logger),
		stopChan:     make(chan struct{}),
		appStore:     appStore,
	}
}

// Stop signals the background scheduler to stop.
func (s *Service) Stop() {
	close(s.stopChan)
}

// StartScheduler starts a background goroutine that backfills missing translations every 30 minutes.
func (s *Service) StartScheduler() {
	go func() {
		// Run immediately on start
		s.BackfillMissingTranslations()

		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.BackfillMissingTranslations()
			case <-s.stopChan:
				slog.Info("Translation scheduler stopped")
				return
			}
		}
	}()
	slog.Info("Translation scheduler started (polling every 30 minutes)")
}

// GetTranslation fetches a translation for a given schematic and language.
// It checks the cache first, then falls back to the database.
func (s *Service) GetTranslation(cacheService *cache.Service, schematicID, lang string) *Translation {
	ck := cache.TranslationKey(schematicID, lang)
	if v, found := cacheService.Get(ck); found {
		if t, ok := v.(*Translation); ok {
			return t
		}
	}

	ctx := context.Background()
	st, err := s.appStore.Translations.GetSchematicTranslation(ctx, schematicID, lang)
	if err != nil || st == nil {
		return nil
	}

	t := &Translation{
		ID:          st.ID,
		Schematic:   schematicID,
		Language:    st.Language,
		Title:       st.Title,
		Description: st.Description,
		Content:     st.Content,
	}
	cacheService.SetWithTTL(ck, t, 60*time.Minute)
	return t
}

// GetTranslationCached is like GetTranslation but uses the internally stored store reference.
func (s *Service) GetTranslationCached(cacheService *cache.Service, schematicID, lang string) *Translation {
	return s.GetTranslation(cacheService, schematicID, lang)
}

// SaveOriginalLanguage stores the original text as a translation record for the detected language.
func (s *Service) SaveOriginalLanguage(schematicID, lang, title, description, content string) error {
	ctx := context.Background()

	// Check if it already exists
	existing, err := s.appStore.Translations.GetSchematicTranslation(ctx, schematicID, lang)
	if err == nil && existing != nil {
		return nil // already saved
	}

	return s.appStore.Translations.UpsertSchematicTranslation(ctx, schematicID, &store.Translation{
		Language:    lang,
		Title:       title,
		Description: description,
		Content:     content,
	})
}

// TranslateAndSave translates all text fields to a target language in a single API call
// and saves the translation record.
func (s *Service) TranslateAndSave(schematicID, targetLang, title, description, content string) error {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return nil
	}

	ctx := context.Background()

	// Check if it already exists
	existing, err := s.appStore.Translations.GetSchematicTranslation(ctx, schematicID, targetLang)
	if err == nil && existing != nil {
		return nil // already translated
	}

	langName := langNames[targetLang]
	if langName == "" {
		langName = targetLang
	}

	fields, err := s.openaiClient.TranslateFields(title, description, content, langName)
	if err != nil {
		slog.Debug("translation failed", "lang", targetLang, "error", err)
		return nil
	}

	// Only save if we got at least a title
	if fields.Title == "" {
		return nil
	}

	return s.appStore.Translations.UpsertSchematicTranslation(ctx, schematicID, &store.Translation{
		Language:    targetLang,
		Title:       fields.Title,
		Description: fields.Description,
		Content:     fields.Content,
	})
}

// TranslateSchematic generates all missing language translations for a single schematic.
func (s *Service) TranslateSchematic(schematicID string) {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return
	}

	ctx := context.Background()
	rec, err := s.appStore.Schematics.GetByID(ctx, schematicID)
	if err != nil {
		return
	}

	title := rec.Title
	description := rec.Description
	content := rec.Content
	detectedLang := rec.DetectedLanguage

	// Save original language if not already saved
	if detectedLang != "" && detectedLang != "en" {
		// The original text might already be overwritten with English at this point,
		// so we check if a record for the detected language already exists
		_ = s.SaveOriginalLanguage(schematicID, detectedLang, title, description, content)
	}

	// Use the English version (stored on the main record) as the source for all translations
	for _, lang := range SupportedLanguages {
		if lang == "en" {
			// Save English version from the main record
			_ = s.SaveOriginalLanguage(schematicID, "en", title, description, content)
			continue
		}
		err := s.TranslateAndSave(schematicID, lang, title, description, content)
		if err != nil {
			slog.Debug("TranslateSchematic: failed to save translation", "id", schematicID, "lang", lang, "error", err)
		}
		// Rate limit: 1 request per second (each TranslateAndSave makes up to 3 API calls)
		time.Sleep(time.Second)
	}
}

// BackfillMissingTranslations finds schematics with fewer than the expected number of
// translation records and generates the missing ones.
func (s *Service) BackfillMissingTranslations() {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		slog.Info("BackfillMissingTranslations skipped: OpenAI not configured")
		return
	}
	slog.Info("BackfillMissingTranslations started")

	ctx := context.Background()
	processed := 0

	for _, lang := range SupportedLanguages {
		if processed >= 20 {
			break
		}

		schematics, err := s.appStore.Translations.ListSchematicsWithoutTranslation(ctx, lang, 20-processed)
		if err != nil {
			slog.Error("BackfillMissingTranslations query failed", "lang", lang, "error", err)
			continue
		}

		for _, schematic := range schematics {
			if processed >= 20 {
				break
			}

			slog.Info("BackfillMissingTranslations: translating schematic", "id", schematic.ID, "lang", lang)
			s.TranslateSchematic(schematic.ID)
			processed++
			time.Sleep(time.Second)
		}
	}

	slog.Info("BackfillMissingTranslations completed", "processed", processed)

	// Also backfill guides and collections
	s.BackfillGuideTranslations()
	s.BackfillCollectionTranslations()
}

// ---------- Guide translations ----------

// GetGuideTranslation fetches a translation for a given guide and language.
func (s *Service) GetGuideTranslation(cacheService *cache.Service, guideID, lang string) *GuideTranslation {
	ck := cache.GuideTranslationKey(guideID, lang)
	if v, found := cacheService.Get(ck); found {
		if t, ok := v.(*GuideTranslation); ok {
			return t
		}
	}

	ctx := context.Background()
	st, err := s.appStore.Translations.GetGuideTranslation(ctx, guideID, lang)
	if err != nil || st == nil {
		return nil
	}

	t := &GuideTranslation{
		ID:          st.ID,
		Guide:       guideID,
		Language:    st.Language,
		Title:       st.Title,
		Description: st.Description,
		Content:     st.Content,
	}
	cacheService.SetWithTTL(ck, t, 60*time.Minute)
	return t
}

// GetGuideTranslationCached is like GetGuideTranslation but uses the internally stored store reference.
func (s *Service) GetGuideTranslationCached(cacheService *cache.Service, guideID, lang string) *GuideTranslation {
	return s.GetGuideTranslation(cacheService, guideID, lang)
}

// TranslateAndSaveGuide translates guide fields to a target language and saves the record.
func (s *Service) TranslateAndSaveGuide(guideID, targetLang, title, description, content string) error {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return nil
	}

	ctx := context.Background()

	existing, err := s.appStore.Translations.GetGuideTranslation(ctx, guideID, targetLang)
	if err == nil && existing != nil {
		return nil
	}

	langName := langNames[targetLang]
	if langName == "" {
		langName = targetLang
	}

	fields, err := s.openaiClient.TranslateFields(title, description, content, langName)
	if err != nil {
		slog.Debug("guide translation failed", "lang", targetLang, "error", err)
		return nil
	}

	if fields.Title == "" {
		return nil
	}

	return s.appStore.Translations.UpsertGuideTranslation(ctx, guideID, &store.Translation{
		Language:    targetLang,
		Title:       fields.Title,
		Description: fields.Description,
		Content:     fields.Content,
	})
}

// TranslateGuide generates all missing language translations for a single guide.
func (s *Service) TranslateGuide(guideID string) {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return
	}

	ctx := context.Background()
	rec, err := s.appStore.Guides.GetByID(ctx, guideID)
	if err != nil {
		return
	}

	title := rec.Title
	description := rec.Excerpt
	content := rec.Content

	for _, lang := range SupportedLanguages {
		if lang == "en" {
			continue
		}
		err := s.TranslateAndSaveGuide(guideID, lang, title, description, content)
		if err != nil {
			slog.Debug("TranslateGuide: failed to save translation", "id", guideID, "lang", lang, "error", err)
		}
		time.Sleep(time.Second)
	}
}

// BackfillGuideTranslations finds guides with fewer than the expected number of
// translation records and generates the missing ones.
func (s *Service) BackfillGuideTranslations() {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return
	}
	slog.Info("BackfillGuideTranslations started")

	ctx := context.Background()

	// Expected translations: all languages minus English (guides are assumed English source)
	expectedCount := len(SupportedLanguages) - 1

	guides, err := s.appStore.Guides.List(ctx, 200, 0)
	if err != nil {
		slog.Error("BackfillGuideTranslations query failed", "error", err)
		return
	}

	processed := 0
	for _, guide := range guides {
		if processed >= 10 {
			break
		}

		// Count existing translations for this guide by checking each language
		count := 0
		for _, lang := range SupportedLanguages {
			if lang == "en" {
				continue
			}
			t, err := s.appStore.Translations.GetGuideTranslation(ctx, guide.ID, lang)
			if err == nil && t != nil {
				count++
			}
		}

		if count >= expectedCount {
			continue
		}

		slog.Info("BackfillGuideTranslations: translating guide", "id", guide.ID, "existing", count, "expected", expectedCount)
		s.TranslateGuide(guide.ID)
		processed++
		time.Sleep(time.Second)
	}

	slog.Info("BackfillGuideTranslations completed", "processed", processed)
}

// ---------- Collection translations ----------

// GetCollectionTranslation fetches a translation for a given collection and language.
func (s *Service) GetCollectionTranslation(cacheService *cache.Service, collectionID, lang string) *CollectionTranslation {
	ck := cache.CollectionTranslationKey(collectionID, lang)
	if v, found := cacheService.Get(ck); found {
		if t, ok := v.(*CollectionTranslation); ok {
			return t
		}
	}

	ctx := context.Background()
	st, err := s.appStore.Translations.GetCollectionTranslation(ctx, collectionID, lang)
	if err != nil || st == nil {
		return nil
	}

	t := &CollectionTranslation{
		ID:          st.ID,
		Collection:  collectionID,
		Language:    st.Language,
		Title:       st.Title,
		Description: st.Description,
	}
	cacheService.SetWithTTL(ck, t, 60*time.Minute)
	return t
}

// GetCollectionTranslationCached is like GetCollectionTranslation but uses the internally stored store reference.
func (s *Service) GetCollectionTranslationCached(cacheService *cache.Service, collectionID, lang string) *CollectionTranslation {
	return s.GetCollectionTranslation(cacheService, collectionID, lang)
}

// TranslateAndSaveCollection translates collection fields to a target language and saves the record.
func (s *Service) TranslateAndSaveCollection(collectionID, targetLang, title, description string) error {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return nil
	}

	ctx := context.Background()

	existing, err := s.appStore.Translations.GetCollectionTranslation(ctx, collectionID, targetLang)
	if err == nil && existing != nil {
		return nil
	}

	langName := langNames[targetLang]
	if langName == "" {
		langName = targetLang
	}

	fields, err := s.openaiClient.TranslateFields(title, description, "", langName)
	if err != nil {
		slog.Debug("collection translation failed", "lang", targetLang, "error", err)
		return nil
	}

	if fields.Title == "" {
		return nil
	}

	return s.appStore.Translations.UpsertCollectionTranslation(ctx, collectionID, &store.Translation{
		Language:    targetLang,
		Title:       fields.Title,
		Description: fields.Description,
	})
}

// TranslateCollection generates all missing language translations for a single collection.
func (s *Service) TranslateCollection(collectionID string) {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return
	}

	ctx := context.Background()
	rec, err := s.appStore.Collections.GetByID(ctx, collectionID)
	if err != nil {
		return
	}

	title := rec.Title
	if title == "" {
		title = rec.Name
	}
	description := rec.Description

	for _, lang := range SupportedLanguages {
		if lang == "en" {
			continue
		}
		err := s.TranslateAndSaveCollection(collectionID, lang, title, description)
		if err != nil {
			slog.Debug("TranslateCollection: failed to save translation", "id", collectionID, "lang", lang, "error", err)
		}
		time.Sleep(time.Second)
	}
}

// BackfillCollectionTranslations finds published collections with fewer than the expected
// number of translation records and generates the missing ones.
func (s *Service) BackfillCollectionTranslations() {
	if s.openaiClient == nil || !s.openaiClient.HasApiKey() {
		return
	}
	slog.Info("BackfillCollectionTranslations started")

	ctx := context.Background()

	expectedCount := len(SupportedLanguages) - 1

	collections, err := s.appStore.Collections.ListPublished(ctx, 200, 0)
	if err != nil {
		slog.Error("BackfillCollectionTranslations query failed", "error", err)
		return
	}

	processed := 0
	for _, coll := range collections {
		if processed >= 10 {
			break
		}

		// Count existing translations for this collection by checking each language
		count := 0
		for _, lang := range SupportedLanguages {
			if lang == "en" {
				continue
			}
			t, err := s.appStore.Translations.GetCollectionTranslation(ctx, coll.ID, lang)
			if err == nil && t != nil {
				count++
			}
		}

		if count >= expectedCount {
			continue
		}

		slog.Info("BackfillCollectionTranslations: translating collection", "id", coll.ID, "existing", count, "expected", expectedCount)
		s.TranslateCollection(coll.ID)
		processed++
		time.Sleep(time.Second)
	}

	slog.Info("BackfillCollectionTranslations completed", "processed", processed)
}
