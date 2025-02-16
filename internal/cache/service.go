package cache

import (
	"createmod/internal/models"
	"fmt"
	"github.com/patrickmn/go-cache"
	"time"
)

// It's never the cache

const (
	AllTagsWithCountKey       = "AllTagsWithCount"
	HighestRatedSchematicsKey = "HighestRatedSchematics"
	TrendingSchematicsKey     = "TrendingSchematics"
	AllCategoriesKey          = "AllCategories"
)

type Service struct {
	c *cache.Cache
}

// New creates the CreateMod.com in-memory cache service
func New() *Service {
	c := cache.New(60*time.Minute, 120*time.Minute)
	return &Service{
		c: c,
	}
}

func SchematicKey(schematicId string) string {
	return fmt.Sprintf("schematic:%s", schematicId)
}

func ViewKey(schematicId string) string {
	return fmt.Sprintf("views:%s", schematicId)
}

func RatingKey(schematicId string) string {
	return fmt.Sprintf("rating:%s", schematicId)
}

func RatingCountKey(schematicId string) string {
	return fmt.Sprintf("ratingCount:%s", schematicId)
}

func (s *Service) Set(key string, value interface{}) {
	s.c.Set(key, value, cache.DefaultExpiration)
}

func (s *Service) Get(key string) (interface{}, bool) {
	return s.c.Get(key)
}

func (s *Service) SetInt(key string, i int) {
	s.c.Set(key, i, cache.DefaultExpiration)
}

func (s *Service) GetInt(key string) (int, bool) {
	v, found := s.Get(key)
	if !found {
		return 0, found
	}
	if i, ok := v.(int); ok {
		return i, found
	}
	return 0, false
}

func (s *Service) SetFloat(key string, f float64) {
	s.c.Set(key, f, cache.DefaultExpiration)
}

func (s *Service) GetFloat(key string) (float64, bool) {
	v, found := s.Get(key)
	if !found {
		return 0, found
	}
	if f, ok := v.(float64); ok {
		return f, found
	}
	return 0, false
}

func (s *Service) SetString(key string, value string) {
	s.c.Set(key, value, cache.DefaultExpiration)
}

func (s *Service) GetString(key string) (string, bool) {
	v, found := s.Get(key)
	if !found {
		return "", found
	}
	if str, ok := v.(string); ok {
		return str, found
	}
	return "", false
}

func (s *Service) SetSchematic(key string, value models.Schematic) {
	s.c.Set(key, value, cache.DefaultExpiration)
}

func (s *Service) DeleteSchematic(key string) {
	s.c.Delete(key)
}

func (s *Service) GetSchematic(key string) (models.Schematic, bool) {
	v, found := s.Get(key)
	if !found {
		return models.Schematic{}, found
	}
	if schem, ok := v.(models.Schematic); ok {
		return schem, found
	}
	return models.Schematic{}, false
}

func (s *Service) SetSchematics(key string, value []models.Schematic) {
	s.c.Set(key, value, cache.DefaultExpiration)
}

func (s *Service) GetSchematics(key string) ([]models.Schematic, bool) {
	v, found := s.Get(key)
	if !found {
		return nil, found
	}
	if schem, ok := v.([]models.Schematic); ok {
		return schem, found
	}
	return nil, false
}

func (s *Service) SetCategories(key string, value []models.SchematicCategory, duration time.Duration) {
	s.c.Set(key, value, duration)
}

func (s *Service) GetCategories(key string) ([]models.SchematicCategory, bool) {
	v, found := s.Get(key)
	if !found {
		return nil, found
	}
	if categories, ok := v.([]models.SchematicCategory); ok {
		return categories, found
	}
	return nil, false
}

func (s *Service) SetTagWithCount(key string, tags []models.SchematicTagWithCount) {
	s.c.Set(key, tags, cache.DefaultExpiration)
}

func (s *Service) GetTagWithCount(key string) ([]models.SchematicTagWithCount, bool) {
	v, found := s.Get(key)
	if !found {
		return nil, found
	}
	if tags, ok := v.([]models.SchematicTagWithCount); ok {
		return tags, found
	}
	return nil, false
}
