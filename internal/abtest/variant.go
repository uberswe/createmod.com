// Package abtest implements multi-variant A/B testing for search backends.
package abtest

// Variant describes a search test variant.
type Variant struct {
	Name       string // "A", "B", "C", "D", "E"
	Engine     string // "bleve" or "meilisearch"
	IndexLevel string // "base", "ai", "full"
	Weight     int    // relative weight for random assignment
}

// DefaultVariants returns the five search test variants with equal weight.
func DefaultVariants() []Variant {
	return []Variant{
		{Name: "A", Engine: "bleve", IndexLevel: "base", Weight: 20},
		{Name: "B", Engine: "bleve", IndexLevel: "ai", Weight: 20},
		{Name: "C", Engine: "meilisearch", IndexLevel: "base", Weight: 20},
		{Name: "D", Engine: "meilisearch", IndexLevel: "ai", Weight: 20},
		{Name: "E", Engine: "meilisearch", IndexLevel: "full", Weight: 20},
	}
}

// VariantByName returns the variant with the given name, or nil.
func VariantByName(variants []Variant, name string) *Variant {
	for i := range variants {
		if variants[i].Name == name {
			return &variants[i]
		}
	}
	return nil
}
