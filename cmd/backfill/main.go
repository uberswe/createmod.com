package main

import (
	"createmod/internal/nbtparser"
	_ "createmod/migrations"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
)

func main() {
	_ = godotenv.Load()

	app := pocketbase.New()

	// Bootstrap the app to run migrations and access the DB
	if err := app.Bootstrap(); err != nil {
		log.Fatal(err)
	}

	records, err := app.FindRecordsByFilter("schematics", "deleted = ''", "-created", -1, 0)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d schematics to process\n", len(records))

	updated := 0
	skipped := 0
	failed := 0

	for _, record := range records {
		// Skip if materials already populated
		if record.GetString("materials") != "" {
			skipped++
			continue
		}

		schematicFile := record.GetString("schematic_file")
		if schematicFile == "" {
			skipped++
			continue
		}

		if !strings.HasSuffix(schematicFile, ".nbt") {
			skipped++
			continue
		}

		// Read the schematic file from pb_data/storage
		filePath := fmt.Sprintf("pb_data/storage/%s/%s", record.BaseFilesPath(), schematicFile)
		f, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("  SKIP %s: cannot open file: %v\n", record.GetString("title"), err)
			failed++
			continue
		}

		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			fmt.Printf("  SKIP %s: cannot read file: %v\n", record.GetString("title"), err)
			failed++
			continue
		}

		materials, err := nbtparser.ExtractMaterials(data)
		if err != nil {
			fmt.Printf("  SKIP %s: cannot extract materials: %v\n", record.GetString("title"), err)
			failed++
			continue
		}

		if len(materials) == 0 {
			skipped++
			continue
		}

		materialsJSON, err := json.Marshal(materials)
		if err != nil {
			failed++
			continue
		}

		record.Set("materials", string(materialsJSON))

		// Extract and set block count
		blockCount, _, _ := nbtparser.ExtractStats(data)
		if blockCount > 0 {
			record.Set("block_count", blockCount)
		}

		// Extract and set dimensions
		dimX, dimY, dimZ, _ := nbtparser.ExtractDimensions(data)
		if dimX > 0 || dimY > 0 || dimZ > 0 {
			record.Set("dim_x", dimX)
			record.Set("dim_y", dimY)
			record.Set("dim_z", dimZ)
		}

		// Extract mod namespaces from materials
		modSet := make(map[string]struct{})
		for _, m := range materials {
			parts := strings.SplitN(m.BlockID, ":", 2)
			if len(parts) == 2 && parts[0] != "minecraft" && parts[0] != "" {
				modSet[parts[0]] = struct{}{}
			}
		}
		if len(modSet) > 0 {
			mods := make([]string, 0, len(modSet))
			for mod := range modSet {
				mods = append(mods, mod)
			}
			modsJSON, err := json.Marshal(mods)
			if err == nil {
				record.Set("mods", string(modsJSON))
			}
		}

		if err := app.Save(record); err != nil {
			fmt.Printf("  FAIL %s: cannot save: %v\n", record.GetString("title"), err)
			failed++
			continue
		}

		updated++
		fmt.Printf("  OK %s: %d materials, %dx%dx%d, %d blocks\n", record.GetString("title"), len(materials), dimX, dimY, dimZ, blockCount)
	}

	fmt.Printf("\nDone! Updated: %d, Skipped: %d, Failed: %d\n", updated, skipped, failed)
}
