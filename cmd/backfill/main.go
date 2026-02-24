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

	records, err := app.FindRecordsByFilter("schematics", "deleted = null", "-created", -1, 0)
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
		if err := app.Save(record); err != nil {
			fmt.Printf("  FAIL %s: cannot save: %v\n", record.GetString("title"), err)
			failed++
			continue
		}

		updated++
		fmt.Printf("  OK %s: %d materials\n", record.GetString("title"), len(materials))
	}

	fmt.Printf("\nDone! Updated: %d, Skipped: %d, Failed: %d\n", updated, skipped, failed)
}
