package schematic

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"io"
	"strings"
	"testing"

	"github.com/Tnze/go-mc/nbt"
	"github.com/Tnze/go-mc/save/region"
)

func writeWorldZip(t *testing.T, s *Schematic) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	warnings, err := WriteWorld(s, "test_world", &buf)
	if err != nil {
		t.Fatalf("WriteWorld: %v", err)
	}
	_ = warnings
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip: %v", err)
	}
	return zr
}

func zipFile(t *testing.T, zr *zip.Reader, name string) []byte {
	t.Helper()
	for _, f := range zr.File {
		if f.Name == name {
			r, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			data, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			return data
		}
	}
	t.Fatalf("zip missing %s (have %v)", name, zr.File)
	return nil
}

func Test_WriteWorld_GoldenStructure(t *testing.T) {
	s, err := ReadStructureNBT(handmadeFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	zr := writeWorldZip(t, s)

	// level.dat: gzipped NBT with sane fields
	levelRaw, err := decompress(zipFile(t, zr, "test_world/level.dat"))
	if err != nil {
		t.Fatalf("level.dat not gzipped NBT: %v", err)
	}
	var level struct {
		Data struct {
			DataVersion int32             `nbt:"DataVersion"`
			LevelName   string            `nbt:"LevelName"`
			GameType    int32             `nbt:"GameType"`
			SpawnY      int32             `nbt:"SpawnY"`
			GameRules   map[string]string `nbt:"GameRules"`
			WGS         nbt.RawMessage    `nbt:"WorldGenSettings"`
		} `nbt:"Data"`
	}
	if err := nbt.Unmarshal(levelRaw, &level); err != nil {
		t.Fatalf("level.dat decode: %v", err)
	}
	if level.Data.DataVersion != 3955 {
		t.Errorf("level DataVersion = %d", level.Data.DataVersion)
	}
	if level.Data.LevelName != "test_world" || level.Data.GameType != 1 {
		t.Errorf("level: name=%q gametype=%d", level.Data.LevelName, level.Data.GameType)
	}
	if level.Data.SpawnY != -60 {
		t.Errorf("SpawnY = %d", level.Data.SpawnY)
	}
	if level.Data.GameRules["doMobSpawning"] != "false" {
		t.Errorf("gamerules: %v", level.Data.GameRules)
	}
	if !strings.Contains(string(level.Data.WGS.Data), "minecraft:flat") {
		t.Errorf("world gen settings missing flat generator")
	}

	// region file: parseable, sector-aligned, chunk (0,0) present
	mca := zipFile(t, zr, "test_world/region/r.0.0.mca")
	if len(mca)%4096 != 0 {
		t.Errorf("region file not sector-aligned: %d bytes", len(mca))
	}
	if len(mca) < 3*4096 {
		t.Errorf("region file too small: %d", len(mca))
	}
	mf := &memFile{buf: mca}
	reg, err := region.Load(mf)
	if err != nil {
		t.Fatalf("region load: %v", err)
	}
	if !reg.ExistSector(0, 0) {
		t.Fatalf("chunk 0,0 missing")
	}
	sector, err := reg.ReadSector(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if sector[0] != 2 {
		t.Fatalf("compression scheme = %d, want 2 (zlib)", sector[0])
	}
	zl, err := zlib.NewReader(bytes.NewReader(sector[1:]))
	if err != nil {
		t.Fatal(err)
	}
	chunkRaw, err := io.ReadAll(zl)
	if err != nil {
		t.Fatal(err)
	}
	var chunk struct {
		DataVersion int32  `nbt:"DataVersion"`
		Status      string `nbt:"Status"`
		XPos        int32  `nbt:"xPos"`
		YPos        int32  `nbt:"yPos"`
		Sections    []struct {
			Y           int8 `nbt:"Y"`
			BlockStates struct {
				Palette []structPaletteEntry `nbt:"palette"`
				Data    []int64              `nbt:"data"`
			} `nbt:"block_states"`
			Biomes struct {
				Palette []string `nbt:"palette"`
			} `nbt:"biomes"`
		} `nbt:"sections"`
		BlockEntities []nbt.RawMessage `nbt:"block_entities"`
	}
	if err := nbt.Unmarshal(chunkRaw, &chunk); err != nil {
		t.Fatalf("chunk decode: %v", err)
	}
	if chunk.Status != "minecraft:full" || chunk.YPos != -4 {
		t.Errorf("chunk: status=%q ypos=%d", chunk.Status, chunk.YPos)
	}

	// The section containing y=-64..-49 must hold the flat layers AND the
	// build blocks (build base at -60 is inside section -4).
	var bottom *struct {
		Y           int8 `nbt:"Y"`
		BlockStates struct {
			Palette []structPaletteEntry `nbt:"palette"`
			Data    []int64              `nbt:"data"`
		} `nbt:"block_states"`
		Biomes struct {
			Palette []string `nbt:"palette"`
		} `nbt:"biomes"`
	}
	for i := range chunk.Sections {
		if chunk.Sections[i].Y == -4 {
			bottom = &chunk.Sections[i]
		}
	}
	if bottom == nil {
		t.Fatalf("no section Y=-4")
	}
	names := map[string]bool{}
	for _, p := range bottom.BlockStates.Palette {
		names[p.Name] = true
	}
	for _, want := range []string{"minecraft:bedrock", "minecraft:dirt", "minecraft:grass_block", "minecraft:stone", "minecraft:oak_stairs", "minecraft:chest"} {
		if !names[want] {
			t.Errorf("section palette missing %s (have %v)", want, names)
		}
	}
	if len(bottom.BlockStates.Data) == 0 {
		t.Errorf("section has no packed data")
	}

	// Block entity carried with world coordinates (chest at build 0,1,0 ->
	// world 0,-59,0)
	if len(chunk.BlockEntities) != 1 {
		t.Fatalf("block entities = %d", len(chunk.BlockEntities))
	}
	var be struct {
		ID string `nbt:"id"`
		X  int32  `nbt:"x"`
		Y  int32  `nbt:"y"`
		Z  int32  `nbt:"z"`
	}
	if err := unmarshalRaw(chunk.BlockEntities[0], &be); err != nil {
		t.Fatal(err)
	}
	if be.ID != "minecraft:chest" || be.Y != -59 {
		t.Errorf("block entity: %+v", be)
	}
}

func Test_WriteWorld_Guards(t *testing.T) {
	if ok, _ := CanExportWorld([3]int{2000, 10, 10}); ok {
		t.Errorf("footprint guard failed")
	}
	if ok, _ := CanExportWorld([3]int{10, 400, 10}); ok {
		t.Errorf("height guard failed")
	}
	if ok, _ := CanExportWorld([3]int{24, 8, 5}); !ok {
		t.Errorf("small build rejected")
	}
	// Old DataVersion clamps with a warning
	s, _ := ReadStructureNBT(handmadeFixture(t))
	s.DataVersion = 1631
	var buf bytes.Buffer
	warnings, err := WriteWorld(s, "w", &buf)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "old or unknown") {
			found = true
		}
	}
	if !found {
		t.Errorf("no clamp warning: %v", warnings)
	}
}

func Test_WriteWorld_MultiChunk(t *testing.T) {
	// A 40x5x40 build spans a 3x3 chunk area; all chunks must exist.
	s := New(40, 5, 40)
	s.DataVersion = 3465
	stone := s.PaletteIndex(BlockState{Name: "minecraft:stone"})
	for i := range s.Blocks {
		s.Blocks[i] = stone
	}
	zr := writeWorldZip(t, s)
	mca := zipFile(t, zr, "test_world/region/r.0.0.mca")
	reg, err := region.Load(&memFile{buf: mca})
	if err != nil {
		t.Fatal(err)
	}
	for cz := 0; cz < 3; cz++ {
		for cx := 0; cx < 3; cx++ {
			if !reg.ExistSector(cx, cz) {
				t.Errorf("chunk %d,%d missing", cx, cz)
			}
		}
	}
	// Status for 1.20.1 build
	sector, _ := reg.ReadSector(2, 2)
	zl, _ := zlib.NewReader(bytes.NewReader(sector[1:]))
	raw, _ := io.ReadAll(zl)
	var chunk struct {
		Status string `nbt:"Status"`
		XPos   int32  `nbt:"xPos"`
	}
	_ = nbt.Unmarshal(raw, &chunk)
	if chunk.Status != "minecraft:full" || chunk.XPos != 2 {
		t.Errorf("chunk 2,2: %+v", chunk)
	}
}
