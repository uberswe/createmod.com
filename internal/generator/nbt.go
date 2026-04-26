package generator

import (
	"bytes"
	"compress/gzip"
	"sort"

	"github.com/Tnze/go-mc/nbt"
)

type nbtPaletteEntry struct {
	Name       string            `nbt:"Name"`
	Properties map[string]string `nbt:"Properties,omitempty"`
}

type nbtBlock struct {
	Pos   [3]int32 `nbt:"pos"`
	State int32    `nbt:"state"`
}

type nbtStructure struct {
	DataVersion int32             `nbt:"DataVersion"`
	Size        [3]int32          `nbt:"size"`
	Palette     []nbtPaletteEntry `nbt:"palette"`
	Blocks      []nbtBlock        `nbt:"blocks"`
	Entities    []interface{}     `nbt:"entities"`
}

type paletteKey struct {
	Name  string
	Props string
}

func blockToPalette(b Block) (string, map[string]string) {
	switch b.Type {
	case BlockWool:
		return "minecraft:white_wool", nil
	case BlockPlank:
		return "minecraft:spruce_planks", nil
	case BlockLog:
		return "minecraft:spruce_log", map[string]string{"axis": "x"}
	case BlockSlabBot:
		return "minecraft:spruce_slab", map[string]string{"type": "bottom", "waterlogged": "false"}
	case BlockSlabTop:
		return "minecraft:spruce_slab", map[string]string{"type": "top", "waterlogged": "false"}
	case BlockStair:
		props := map[string]string{
			"facing":      "north",
			"half":        "bottom",
			"shape":       "straight",
			"waterlogged": "false",
		}
		if b.Props != nil {
			for k, v := range b.Props {
				props[k] = v
			}
		}
		return "minecraft:spruce_stairs", props
	case BlockFence:
		return "minecraft:spruce_fence", map[string]string{
			"east":        "false",
			"north":       "false",
			"south":       "false",
			"waterlogged": "false",
			"west":        "false",
		}
	case BlockTrapdoor:
		props := map[string]string{
			"facing":      "north",
			"half":        "bottom",
			"open":        "false",
			"powered":     "false",
			"waterlogged": "false",
		}
		if b.Props != nil {
			for k, v := range b.Props {
				props[k] = v
			}
		}
		return "minecraft:spruce_trapdoor", props
	default:
		return "minecraft:air", nil
	}
}

func propsKey(props map[string]string) string {
	if len(props) == 0 {
		return ""
	}
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(props[k])
	}
	return buf.String()
}

func ExportNBT(result *GenerateResult) ([]byte, error) {
	paletteMap := make(map[paletteKey]int32)
	var palette []nbtPaletteEntry

	getPaletteIndex := func(name string, props map[string]string) int32 {
		pk := paletteKey{Name: name, Props: propsKey(props)}
		if idx, ok := paletteMap[pk]; ok {
			return idx
		}
		idx := int32(len(palette))
		paletteMap[pk] = idx
		palette = append(palette, nbtPaletteEntry{Name: name, Properties: props})
		return idx
	}

	var nbtBlocks []nbtBlock
	for _, b := range result.Blocks {
		name, props := blockToPalette(b)
		if name == "minecraft:air" {
			continue
		}
		idx := getPaletteIndex(name, props)
		nbtBlocks = append(nbtBlocks, nbtBlock{
			Pos:   [3]int32{int32(b.X), int32(b.Y), int32(b.Z)},
			State: idx,
		})
	}

	structure := nbtStructure{
		DataVersion: 3955,
		Size:        [3]int32{int32(result.SizeX), int32(result.SizeY), int32(result.SizeZ)},
		Palette:     palette,
		Blocks:      nbtBlocks,
		Entities:    []interface{}{},
	}

	var rawBuf bytes.Buffer
	if err := nbt.NewEncoder(&rawBuf).Encode(structure, ""); err != nil {
		return nil, err
	}

	var gzBuf bytes.Buffer
	gz := gzip.NewWriter(&gzBuf)
	if _, err := gz.Write(rawBuf.Bytes()); err != nil {
		gz.Close()
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	return gzBuf.Bytes(), nil
}
