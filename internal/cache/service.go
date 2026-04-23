package cache

import (
	"context"
	"createmod/internal/models"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
)

// It's never the cache

const (
	AllTagsWithCountKey       = "AllTagsWithCount"
	HighestRatedSchematicsKey = "HighestRatedSchematics"
	TrendingSchematicsKey     = "TrendingSchematics"
	AllCategoriesKey          = "AllCategories"
	LatestSchematicsKey       = "LatestSchematics"
	LatestHasNextKey          = "LatestHasNext"
	HighestRatedHasNextKey    = "HighestRatedHasNext"
	TrendingHasNextKey        = "TrendingHasNext"

	keyPrefix    = "cm:"
	defaultTTL   = 60 * time.Minute
	redisTimeout = 2 * time.Second
)

// TrendingKeyForWindow returns a cache key for trending schematics with a specific time window.
func TrendingKeyForWindow(days int) string {
	return fmt.Sprintf("TrendingSchematics:%d", days)
}

// TrendingHasNextKeyForWindow returns a cache key for the trending hasNext flag with a specific window.
func TrendingHasNextKeyForWindow(days int) string {
	return fmt.Sprintf("TrendingHasNext:%d", days)
}

// CategorySectionKeyForWindow returns a cache key for a category section with a specific window.
func CategorySectionKeyForWindow(catKey string, days int) string {
	return fmt.Sprintf("CategorySection:%s:%d", catKey, days)
}

// CategorySectionHasNextKeyForWindow returns a cache key for a category section hasNext flag with a specific window.
func CategorySectionHasNextKeyForWindow(catKey string, days int) string {
	return fmt.Sprintf("CategorySectionHasNext:%s:%d", catKey, days)
}

// redisEntry is the typed JSON envelope stored in Redis.
type redisEntry struct {
	Type  string          `json:"t"`
	Value json.RawMessage `json:"v"`
}

type Service struct {
	c      *gocache.Cache   // in-memory (always present)
	redis  *redis.Client    // nil if Redis not configured
	pubsub *redis.PubSub   // invalidation subscription
	stopCh chan struct{}    // stops subscription goroutine
}

// SetWithTTL sets a key with a specific TTL duration.
func (s *Service) SetWithTTL(key string, value interface{}, duration time.Duration) {
	s.c.Set(key, value, duration)
}

// Delete removes a key from the cache and publishes invalidation to other pods.
func (s *Service) Delete(key string) {
	s.c.Delete(key)
	if s.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
		defer cancel()
		s.redis.Del(ctx, keyPrefix+key)
		s.publishInvalidation(invalidationMsg{Action: "delete", Keys: []string{key}})
	}
}

// New creates the CreateMod.com in-memory cache service.
func New() *Service {
	c := gocache.New(defaultTTL, 120*time.Minute)
	return &Service{
		c: c,
	}
}

// NewWithRedis creates a cache service backed by both in-memory and Redis.
// Writes go to both; reads try Redis first, fall back to in-memory.
// A pub/sub subscription is started for cross-pod invalidation.
func NewWithRedis(client *redis.Client) *Service {
	c := gocache.New(defaultTTL, 120*time.Minute)
	s := &Service{
		c:      c,
		redis:  client,
		stopCh: make(chan struct{}),
	}
	s.startSubscription()
	return s
}

// Close stops the pub/sub subscription goroutine.
func (s *Service) Close() {
	if s.stopCh != nil {
		close(s.stopCh)
	}
	if s.pubsub != nil {
		_ = s.pubsub.Close()
	}
}

func SchematicKey(schematicId string) string {
	return fmt.Sprintf("schematic:%s", schematicId)
}

func ViewKey(schematicId string) string {
	return fmt.Sprintf("views:%s", schematicId)
}

func DownloadKey(schematicId string) string {
	return fmt.Sprintf("downloads:%s", schematicId)
}

func RatingKey(schematicId string) string {
	return fmt.Sprintf("rating:%s", schematicId)
}

func RatingCountKey(schematicId string) string {
	return fmt.Sprintf("ratingCount:%s", schematicId)
}

func TranslationKey(schematicId, lang string) string {
	return fmt.Sprintf("translation:%s:%s", schematicId, lang)
}

func GuideTranslationKey(guideId, lang string) string {
	return fmt.Sprintf("guide_translation:%s:%s", guideId, lang)
}

func CollectionTranslationKey(collectionId, lang string) string {
	return fmt.Sprintf("collection_translation:%s:%s", collectionId, lang)
}

func CommentTranslationKey(commentId, lang string) string {
	return fmt.Sprintf("comment_translation:%s:%s", commentId, lang)
}

func MinecraftVersionKey(id string) string {
	return fmt.Sprintf("mcversion:%s", id)
}

func CreatemodVersionKey(id string) string {
	return fmt.Sprintf("cmversion:%s", id)
}

func ModMetadataKey(namespace string) string {
	return fmt.Sprintf("modmeta:%s", namespace)
}

func SchematicHTMLKey(name, lang string) string {
	return fmt.Sprintf("html:schematic:%s:%s", name, lang)
}

func SchematicsListHTMLKey(page int, lang string) string {
	return fmt.Sprintf("html:schematics:%d:%s", page, lang)
}

// --- Generic Set/Get (in-memory only for complex types) ---

func (s *Service) Set(key string, value interface{}) {
	s.c.Set(key, value, gocache.DefaultExpiration)
}

func (s *Service) Get(key string) (interface{}, bool) {
	return s.c.Get(key)
}

// --- Typed setters/getters with Redis support ---

func (s *Service) SetInt(key string, i int) {
	s.c.Set(key, i, gocache.DefaultExpiration)
	s.redisSet(key, "int", i)
}

func (s *Service) GetInt(key string) (int, bool) {
	// Try Redis first
	if s.redis != nil {
		if v, ok := s.redisGetTyped(key, "int"); ok {
			var i int
			if err := json.Unmarshal(v, &i); err == nil {
				s.c.Set(key, i, gocache.DefaultExpiration)
				return i, true
			}
		}
	}
	v, found := s.c.Get(key)
	if !found {
		return 0, false
	}
	if i, ok := v.(int); ok {
		return i, true
	}
	return 0, false
}

func (s *Service) SetFloat(key string, f float64) {
	s.c.Set(key, f, gocache.DefaultExpiration)
	s.redisSet(key, "float", f)
}

func (s *Service) GetFloat(key string) (float64, bool) {
	if s.redis != nil {
		if v, ok := s.redisGetTyped(key, "float"); ok {
			var f float64
			if err := json.Unmarshal(v, &f); err == nil {
				s.c.Set(key, f, gocache.DefaultExpiration)
				return f, true
			}
		}
	}
	v, found := s.c.Get(key)
	if !found {
		return 0, false
	}
	if f, ok := v.(float64); ok {
		return f, true
	}
	return 0, false
}

func (s *Service) SetString(key string, value string) {
	s.c.Set(key, value, gocache.DefaultExpiration)
	s.redisSet(key, "string", value)
}

func (s *Service) GetString(key string) (string, bool) {
	if s.redis != nil {
		if v, ok := s.redisGetTyped(key, "string"); ok {
			var str string
			if err := json.Unmarshal(v, &str); err == nil {
				s.c.Set(key, str, gocache.DefaultExpiration)
				return str, true
			}
		}
	}
	v, found := s.c.Get(key)
	if !found {
		return "", false
	}
	if str, ok := v.(string); ok {
		return str, true
	}
	return "", false
}

func (s *Service) SetSchematic(key string, value models.Schematic) {
	s.c.Set(key, value, gocache.DefaultExpiration)
	s.redisSet(key, "schematic", value)
}

var htmlCacheLanguages = []string{"en", "fr", "pt-BR", "pt-PT", "es", "de", "pl", "ru", "zh-Hans"}

func (s *Service) DeleteSchematicHTML(name string) {
	for _, lang := range htmlCacheLanguages {
		s.c.Delete(SchematicHTMLKey(name, lang))
	}
}

func (s *Service) DeleteSchematicsListHTML() {
	for page := 1; page <= 20; page++ {
		for _, lang := range htmlCacheLanguages {
			s.c.Delete(SchematicsListHTMLKey(page, lang))
		}
	}
}

func (s *Service) DeleteSchematic(key string) {
	s.Delete(key)
}

func (s *Service) GetSchematic(key string) (models.Schematic, bool) {
	if s.redis != nil {
		if v, ok := s.redisGetTyped(key, "schematic"); ok {
			var schem models.Schematic
			if err := json.Unmarshal(v, &schem); err == nil {
				s.c.Set(key, schem, gocache.DefaultExpiration)
				return schem, true
			}
		}
	}
	v, found := s.c.Get(key)
	if !found {
		return models.Schematic{}, false
	}
	if schem, ok := v.(models.Schematic); ok {
		return schem, true
	}
	return models.Schematic{}, false
}

func (s *Service) SetSchematics(key string, value []models.Schematic) {
	s.c.Set(key, value, gocache.DefaultExpiration)
	s.redisSet(key, "schematics", value)
}

func (s *Service) GetSchematics(key string) ([]models.Schematic, bool) {
	if s.redis != nil {
		if v, ok := s.redisGetTyped(key, "schematics"); ok {
			var schem []models.Schematic
			if err := json.Unmarshal(v, &schem); err == nil {
				s.c.Set(key, schem, gocache.DefaultExpiration)
				return schem, true
			}
		}
	}
	v, found := s.c.Get(key)
	if !found {
		return nil, false
	}
	if schem, ok := v.([]models.Schematic); ok {
		return schem, true
	}
	return nil, false
}

func (s *Service) SetCategories(key string, value []models.SchematicCategory, duration time.Duration) {
	s.c.Set(key, value, duration)
	s.redisSetWithTTL(key, "categories", value, duration)
}

func (s *Service) GetCategories(key string) ([]models.SchematicCategory, bool) {
	if s.redis != nil {
		if v, ok := s.redisGetTyped(key, "categories"); ok {
			var categories []models.SchematicCategory
			if err := json.Unmarshal(v, &categories); err == nil {
				s.c.Set(key, categories, gocache.DefaultExpiration)
				return categories, true
			}
		}
	}
	v, found := s.c.Get(key)
	if !found {
		return nil, false
	}
	if categories, ok := v.([]models.SchematicCategory); ok {
		return categories, true
	}
	return nil, false
}

func (s *Service) SetTagWithCount(key string, tags []models.SchematicTagWithCount) {
	s.c.Set(key, tags, gocache.DefaultExpiration)
	s.redisSet(key, "tagswithcount", tags)
}

func (s *Service) GetTagWithCount(key string) ([]models.SchematicTagWithCount, bool) {
	if s.redis != nil {
		if v, ok := s.redisGetTyped(key, "tagswithcount"); ok {
			var tags []models.SchematicTagWithCount
			if err := json.Unmarshal(v, &tags); err == nil {
				s.c.Set(key, tags, gocache.DefaultExpiration)
				return tags, true
			}
		}
	}
	v, found := s.c.Get(key)
	if !found {
		return nil, false
	}
	if tags, ok := v.([]models.SchematicTagWithCount); ok {
		return tags, true
	}
	return nil, false
}

func (s *Service) Flush() {
	s.c.Flush()
	if s.redis != nil {
		// Don't flush Redis entirely — rate limiter shares it.
		// Only publish invalidation so other pods flush their in-memory caches.
		s.publishInvalidation(invalidationMsg{Action: "flush"})
	}
}

// --- Redis helpers ---

// redisSet stores a typed value in Redis with the default TTL.
func (s *Service) redisSet(key string, typeName string, value interface{}) {
	if s.redis == nil {
		return
	}
	s.redisSetWithTTL(key, typeName, value, defaultTTL)
}

// redisSetWithTTL stores a typed value in Redis with a specific TTL.
func (s *Service) redisSetWithTTL(key string, typeName string, value interface{}, ttl time.Duration) {
	if s.redis == nil {
		return
	}
	valBytes, err := json.Marshal(value)
	if err != nil {
		slog.Debug("cache: redis marshal error", "key", key, "error", err)
		return
	}
	entry := redisEntry{Type: typeName, Value: valBytes}
	data, err := json.Marshal(entry)
	if err != nil {
		slog.Debug("cache: redis envelope marshal error", "key", key, "error", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()
	if err := s.redis.Set(ctx, keyPrefix+key, data, ttl).Err(); err != nil {
		slog.Debug("cache: redis SET error", "key", key, "error", err)
	}
}

// redisGetTyped retrieves a value from Redis and returns the raw JSON value
// if the type discriminator matches.
func (s *Service) redisGetTyped(key string, expectedType string) (json.RawMessage, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()
	data, err := s.redis.Get(ctx, keyPrefix+key).Bytes()
	if err != nil {
		return nil, false
	}
	var entry redisEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}
	if entry.Type != expectedType {
		return nil, false
	}
	return entry.Value, true
}
