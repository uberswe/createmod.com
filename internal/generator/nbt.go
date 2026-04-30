package generator

import (
	"bytes"
	"compress/gzip"
	"fmt"
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

func woodPrefix(mat MaterialConfig) string {
	w := mat.WoodType
	if w == "" || !isValidWoodType(w) {
		w = "spruce"
	}
	if w == "crimson" || w == "warped" {
		return "minecraft:" + w
	}
	return "minecraft:" + w
}

func woodPlanks(mat MaterialConfig) string {
	w := mat.WoodType
	if w == "" || !isValidWoodType(w) {
		w = "spruce"
	}
	if w == "crimson" || w == "warped" {
		return fmt.Sprintf("minecraft:%s_planks", w)
	}
	return fmt.Sprintf("minecraft:%s_planks", w)
}

func woodLog(mat MaterialConfig) string {
	w := mat.WoodType
	if w == "" || !isValidWoodType(w) {
		w = "spruce"
	}
	switch w {
	case "crimson":
		return "minecraft:crimson_stem"
	case "warped":
		return "minecraft:warped_stem"
	default:
		return fmt.Sprintf("minecraft:%s_log", w)
	}
}

func woodSlab(mat MaterialConfig) string {
	w := mat.WoodType
	if w == "" || !isValidWoodType(w) {
		w = "spruce"
	}
	return fmt.Sprintf("minecraft:%s_slab", w)
}

func woodStairs(mat MaterialConfig) string {
	w := mat.WoodType
	if w == "" || !isValidWoodType(w) {
		w = "spruce"
	}
	return fmt.Sprintf("minecraft:%s_stairs", w)
}

func woodFence(mat MaterialConfig) string {
	w := mat.WoodType
	if w == "" || !isValidWoodType(w) {
		w = "spruce"
	}
	return fmt.Sprintf("minecraft:%s_fence", w)
}

func woodTrapdoor(mat MaterialConfig) string {
	w := mat.WoodType
	if w == "" || !isValidWoodType(w) {
		w = "spruce"
	}
	return fmt.Sprintf("minecraft:%s_trapdoor", w)
}

func woolBlock(color string) string {
	if color == "" || !isValidWoolColor(color) {
		color = "white"
	}
	return fmt.Sprintf("minecraft:%s_wool", color)
}

func envelopeBlock(color string) string {
	if color == "" || !isValidWoolColor(color) {
		color = "white"
	}
	return fmt.Sprintf("aeronautics:%s_envelope", color)
}

func sailBlock(color string) string {
	if color == "" || !isValidWoolColor(color) {
		color = "white"
	}
	return fmt.Sprintf("create:%s_sail", color)
}

func blockToPalette(b Block, mat MaterialConfig) (string, map[string]string) {
	switch b.Type {
	case BlockWool:
		if mat.EnvelopeMaterial == "envelope" {
			return envelopeBlock(mat.EnvelopeColor), nil
		}
		color := mat.EnvelopeColor
		if color == "" {
			color = mat.BladeColor
		}
		return woolBlock(color), nil

	case BlockSail:
		if mat.BladeMaterial == "sail" {
			return sailBlock(mat.BladeColor), nil
		}
		return woolBlock(mat.BladeColor), nil

	case BlockPlank:
		return woodPlanks(mat), nil

	case BlockLog:
		if mat.FrameMaterial == "andesite_casing" {
			return "create:andesite_casing", nil
		}
		return woodLog(mat), map[string]string{"axis": "x"}

	case BlockSlabBot:
		return woodSlab(mat), map[string]string{"type": "bottom", "waterlogged": "false"}

	case BlockSlabTop:
		return woodSlab(mat), map[string]string{"type": "top", "waterlogged": "false"}

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
		return woodStairs(mat), props

	case BlockFence:
		props := map[string]string{
			"east":        "false",
			"north":       "false",
			"south":       "false",
			"waterlogged": "false",
			"west":        "false",
		}
		if b.Props != nil {
			for k, v := range b.Props {
				props[k] = v
			}
		}
		return woodFence(mat), props

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
		return woodTrapdoor(mat), props

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
	mat := result.Materials

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
		name, props := blockToPalette(b, mat)
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
