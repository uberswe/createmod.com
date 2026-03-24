package search

import (
	"testing"
)

// Compile-time interface check.
var _ SearchEngine = (*MeiliEngine)(nil)

func TestMeiliEngine_BuildFilter(t *testing.T) {
	m := &MeiliEngine{}
	tests := []struct {
		name   string
		query  SearchQuery
		expect string
	}{
		{
			name:   "empty",
			query:  SearchQuery{Category: "all", Rating: -1},
			expect: "",
		},
		{
			name:   "category",
			query:  SearchQuery{Category: "automation", Rating: -1},
			expect: `categories = "automation"`,
		},
		{
			name:   "category with hyphen",
			query:  SearchQuery{Category: "mob-farms", Rating: -1},
			expect: `categories = "mob farms"`,
		},
		{
			name:   "rating",
			query:  SearchQuery{Category: "all", Rating: 3},
			expect: "rating >= 3",
		},
		{
			name:   "tags AND logic",
			query:  SearchQuery{Category: "all", Rating: -1, Tags: []string{"redstone", "compact"}},
			expect: `tags = "redstone" AND tags = "compact"`,
		},
		{
			name:   "minecraft version",
			query:  SearchQuery{Category: "all", Rating: -1, MinecraftVersion: "1.20.1"},
			expect: `minecraft_version = "1.20.1"`,
		},
		{
			name:   "create version",
			query:  SearchQuery{Category: "all", Rating: -1, CreateVersion: "0.5.1"},
			expect: `create_version = "0.5.1"`,
		},
		{
			name:   "block count range",
			query:  SearchQuery{Category: "all", Rating: -1, MinBlockCount: 10, MaxBlockCount: 500},
			expect: "block_count >= 10 AND block_count <= 500",
		},
		{
			name:   "dimension filter",
			query:  SearchQuery{Category: "all", Rating: -1, MinDimY: 5, MaxDimY: 100},
			expect: "dim_y >= 5 AND dim_y <= 100",
		},
		{
			name:   "mod filter",
			query:  SearchQuery{Category: "all", Rating: -1, Mods: []string{"Create", "Minecraft"}},
			expect: `mod_names = "Create" AND mod_names = "Minecraft"`,
		},
		{
			name:   "combined",
			query:  SearchQuery{Category: "automation", Rating: 3, Tags: []string{"redstone"}, MinecraftVersion: "1.20.1"},
			expect: `rating >= 3 AND categories = "automation" AND tags = "redstone" AND minecraft_version = "1.20.1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.buildFilter(tt.query)
			if got != tt.expect {
				t.Errorf("buildFilter() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestMeiliEngine_BuildSort(t *testing.T) {
	m := &MeiliEngine{}
	tests := []struct {
		order  int
		expect string
	}{
		{BestMatchOrder, ""},
		{NewestOrder, "created_timestamp:desc"},
		{OldestOrder, "created_timestamp:asc"},
		{HighestRatingOrder, "rating:desc"},
		{LowestRatingOrder, "rating:asc"},
		{MostViewedOrder, "views:desc"},
		{LeastViewedOrder, "views:asc"},
		{TrendingOrder, ""},
	}

	for _, tt := range tests {
		sort := m.buildSort(tt.order)
		var got string
		if len(sort) > 0 {
			got = sort[0]
		}
		if got != tt.expect {
			t.Errorf("buildSort(%d) = %q, want %q", tt.order, got, tt.expect)
		}
	}
}
