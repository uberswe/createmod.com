package abtest

import (
	"os"
	"strconv"
	"strings"
)

// TrendingConfig holds trending A/B test configuration parsed from environment variables.
type TrendingConfig struct {
	Enabled  bool
	Variants []TrendingVariant
}

// LoadTrendingConfig reads trending A/B test settings from environment variables.
//
//	TRENDING_TEST_ENABLED   — "true" to enable (default: false)
//	TRENDING_TEST_VARIANTS  — "T7:25,T15:25,T30:25,T60:25" (default: equal weights)
func LoadTrendingConfig() *TrendingConfig {
	cfg := &TrendingConfig{
		Enabled: os.Getenv("TRENDING_TEST_ENABLED") == "true",
	}

	raw := os.Getenv("TRENDING_TEST_VARIANTS")
	if raw != "" {
		cfg.Variants = parseTrendingVariants(raw)
	}
	if len(cfg.Variants) == 0 {
		cfg.Variants = DefaultTrendingVariants()
	}
	return cfg
}

// AllWindowDays returns a deduplicated slice of all variant window days.
func (c *TrendingConfig) AllWindowDays() []int {
	seen := make(map[int]bool, len(c.Variants))
	var days []int
	for _, v := range c.Variants {
		if !seen[v.WindowDays] {
			seen[v.WindowDays] = true
			days = append(days, v.WindowDays)
		}
	}
	return days
}

// parseTrendingVariants parses "T1:25,T3:25,T6:25,T12:25" into TrendingVariant slices.
func parseTrendingVariants(raw string) []TrendingVariant {
	defaults := DefaultTrendingVariants()
	byName := make(map[string]*TrendingVariant, len(defaults))
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
	result := make([]TrendingVariant, 0, len(defaults))
	for _, v := range defaults {
		if v.Weight > 0 {
			result = append(result, v)
		}
	}
	return result
}
