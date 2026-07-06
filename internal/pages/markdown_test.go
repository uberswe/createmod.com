package pages

import (
	"strings"
	"testing"

	"createmod/internal/models"
	"createmod/internal/nbtparser"
)

func Test_SchematicMarkdown_Includes_Curated_Content(t *testing.T) {
	d := SchematicData{}
	d.Language = "en"
	d.Schematic = models.Schematic{
		Title:            "Super Farm",
		Name:             "super-farm",
		Content:          "<p>An automatic wheat farm.</p>",
		Rating:           "4.5",
		RatingCount:      12,
		HasRating:        true,
		Views:            3400,
		Downloads:        800,
		MinecraftVersion: "1.20.1",
		CreatemodVersion: "0.5.1",
		BlockCount:       450,
		DimX:             10, DimY: 12, DimZ: 14,
		Mods:          []string{"create", "flywheel"},
		Video:         "https://www.youtube.com/watch?v=abc123DEF45",
		Author:        &models.User{Username: "alice"},
		SchematicFile: "superfarm_v2.nbt",
	}
	d.Materials = []nbtparser.Material{
		{BlockID: "minecraft:oak_planks", Count: 120},
		{BlockID: "create:cogwheel", Count: 32},
	}

	md := schematicMarkdown(d)

	for _, want := range []string{
		"# Super Farm",
		"An automatic wheat farm.",
		"Rating: 4.5/5 (12 ratings)",
		"https://www.youtube.com/watch?v=abc123DEF45",
		"minecraft:oak_planks: 120",
		"create:cogwheel: 32",
		"Required mods: create, flywheel",
		"Dimensions: 10 x 12 x 14",
		"https://createmod.com/schematics/super-farm",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q", want)
		}
	}
}

func Test_SchematicMarkdown_Never_Exposes_Schematic_Files(t *testing.T) {
	d := SchematicData{}
	d.Language = "en"
	d.Schematic = models.Schematic{
		Title:         "Super Farm",
		Name:          "super-farm",
		SchematicFile: "superfarm_v2.nbt",
		Gallery:       []string{"img1.png"},
	}

	md := strings.ToLower(schematicMarkdown(d))
	for _, banned := range []string{
		"superfarm_v2", // the schematic file name
		".nbt)",        // any nbt link
		"/get/",        // download interstitial
		"/api/files/",  // raw file serving route
		"download-url", // tokenized download endpoint
	} {
		if strings.Contains(md, banned) {
			t.Errorf("markdown must not contain %q", banned)
		}
	}
}

func Test_IndexMarkdown_Lists_Rails(t *testing.T) {
	d := IndexData{
		Trending: []models.Schematic{
			{Title: "Cool Train", Name: "cool-train", Rating: "5.0", RatingCount: 3, HasRating: true},
		},
		Schematics:   []models.Schematic{{Title: "New Build", Name: "new-build"}},
		HighestRated: []models.Schematic{{Title: "Top Build", Name: "top-build"}},
	}
	d.Language = "en"

	md := indexMarkdown(d)
	for _, want := range []string{
		"[Cool Train](https://createmod.com/schematics/cool-train)",
		"5.0/5",
		"[New Build](https://createmod.com/schematics/new-build)",
		"[Top Build](https://createmod.com/schematics/top-build)",
		"/.well-known/api-catalog",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("index markdown missing %q", want)
		}
	}
}
