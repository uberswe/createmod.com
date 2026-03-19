package abtest

// TrendingVariant represents a trending time-window A/B test variant.
type TrendingVariant struct {
	Name       string
	WindowDays int
	Weight     int
}

// DefaultTrendingVariants returns the four default trending variants with equal weights.
func DefaultTrendingVariants() []TrendingVariant {
	return []TrendingVariant{
		{Name: "T7", WindowDays: 7, Weight: 25},
		{Name: "T15", WindowDays: 15, Weight: 25},
		{Name: "T30", WindowDays: 30, Weight: 25},
		{Name: "T60", WindowDays: 60, Weight: 25},
	}
}

// TrendingVariantByName returns a pointer to the variant with the given name, or nil.
func TrendingVariantByName(variants []TrendingVariant, name string) *TrendingVariant {
	for i := range variants {
		if variants[i].Name == name {
			return &variants[i]
		}
	}
	return nil
}
