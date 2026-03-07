package main

import (
	"createmod/internal/nbtparser"
	"fmt"
	"os"

	mc "github.com/uberswe/mcnbt"
)

func main() {
	data, err := os.ReadFile("beginner_iron_farm_0NM1tSfTSu.nbt")
	if err != nil {
		fmt.Println("read error:", err)
		return
	}
	fmt.Println("File size:", len(data))

	// Test Validate
	ok, reason := nbtparser.Validate(data)
	fmt.Println("Validate:", ok, reason)

	// Test DecodeAny
	decoded, err := mc.DecodeAny(data)
	fmt.Printf("DecodeAny error: %v\n", err)
	fmt.Printf("DecodeAny type: %T\n", decoded)

	// Test ConvertToStandard
	if err == nil {
		std, err2 := mc.ConvertToStandard(decoded)
		fmt.Printf("ConvertToStandard error: %v\n", err2)
		if err2 == nil && std != nil {
			fmt.Printf("Size: %d x %d x %d\n", std.Size.X, std.Size.Y, std.Size.Z)
			fmt.Printf("Blocks: %d\n", len(std.Blocks))
			fmt.Printf("Palette entries: %d\n", len(std.Palette))
			for k, v := range std.Palette {
				if k < 10 {
					fmt.Printf("  [%d] %s\n", k, v.Name)
				}
			}
		}
	}

	// Test ExtractDimensions
	x, y, z, dimOk := nbtparser.ExtractDimensions(data)
	fmt.Printf("ExtractDimensions: %d x %d x %d (ok=%v)\n", x, y, z, dimOk)

	// Test ExtractMaterials
	mats, matErr := nbtparser.ExtractMaterials(data)
	fmt.Printf("ExtractMaterials error: %v\n", matErr)
	fmt.Printf("Materials count: %d\n", len(mats))
	for i, m := range mats {
		if i < 5 {
			fmt.Printf("  %s: %d\n", m.BlockID, m.Count)
		}
	}

	// Test ExtractStats
	bc, legacyMats, statsOk := nbtparser.ExtractStats(data)
	fmt.Printf("ExtractStats: blockCount=%d, materials=%d, ok=%v\n", bc, len(legacyMats), statsOk)
}
