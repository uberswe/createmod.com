package abtest

import (
	"os"
	"strconv"
	"strings"
)

// Config holds A/B test configuration parsed from environment variables.
type Config struct {
	Enabled        bool
	Variants       []Variant
	MeilisearchURL string
	MeilisearchKey string
}

// LoadConfig reads A/B test settings from environment variables.
//
//	SEARCH_TEST_ENABLED   — "true" to enable (default: false)
//	SEARCH_TEST_VARIANTS  — "A:20,B:20,C:20,D:20,E:20" (default: equal weights)
//	MEILISEARCH_URL       — Meilisearch endpoint (e.g. "http://meilisearch:7700")
//	MEILISEARCH_API_KEY   — Meilisearch master/API key
func LoadConfig() *Config {
	cfg := &Config{
		Enabled:        os.Getenv("SEARCH_TEST_ENABLED") == "true",
		MeilisearchURL: os.Getenv("MEILISEARCH_URL"),
		MeilisearchKey: os.Getenv("MEILISEARCH_API_KEY"),
	}

	raw := os.Getenv("SEARCH_TEST_VARIANTS")
	if raw != "" {
		cfg.Variants = parseVariants(raw)
	}
	if len(cfg.Variants) == 0 {
		cfg.Variants = DefaultVariants()
	}
	return cfg
}

// parseVariants parses "A:20,B:20,C:20,D:20,E:20" into Variant slices,
// updating the weights of the default variants.
func parseVariants(raw string) []Variant {
	defaults := DefaultVariants()
	byName := make(map[string]*Variant, len(defaults))
	for i := range defaults {
		byName[defaults[i].Name] = &defaults[i]
	}

	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pieces := strings.SplitN(part, ":", 2)
		name := strings.TrimSpace(pieces[0])
		v, ok := byName[name]
		if !ok {
			continue
		}
		if len(pieces) == 2 {
			if w, err := strconv.Atoi(strings.TrimSpace(pieces[1])); err == nil && w >= 0 {
				v.Weight = w
			}
		}
	}

	// Return only variants with positive weight.
	result := make([]Variant, 0, len(defaults))
	for _, v := range defaults {
		if v.Weight > 0 {
			result = append(result, v)
		}
	}
	return result
}
