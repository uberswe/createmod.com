package render

import (
	"image/color"

	"createmod/internal/generator"
)

type woodColors struct {
	plank color.RGBA
	log   color.RGBA
}

var woodPalette = map[string]woodColors{
	"oak":      {rgb(0xb8, 0x94, 0x5f), rgb(0x6b, 0x58, 0x39)},
	"spruce":   {rgb(0x6b, 0x42, 0x26), rgb(0x3a, 0x27, 0x18)},
	"birch":    {rgb(0xd5, 0xc9, 0x8c), rgb(0xd5, 0xcd, 0xa1)},
	"dark_oak": {rgb(0x3e, 0x29, 0x12), rgb(0x38, 0x2a, 0x15)},
	"jungle":   {rgb(0xb8, 0x87, 0x64), rgb(0x56, 0x4a, 0x2e)},
	"acacia":   {rgb(0xa8, 0x56, 0x32), rgb(0x67, 0x61, 0x57)},
	"cherry":   {rgb(0xe8, 0xc4, 0xb8), rgb(0x3b, 0x20, 0x22)},
	"crimson":  {rgb(0x6b, 0x33, 0x44), rgb(0x5c, 0x21, 0x33)},
	"warped":   {rgb(0x2b, 0x6b, 0x5e), rgb(0x3a, 0x3f, 0x55)},
}

var woolPalette = map[string]color.RGBA{
	"white":      rgb(0xe8, 0xe8, 0xe8),
	"orange":     rgb(0xf0, 0x76, 0x13),
	"magenta":    rgb(0xbd, 0x44, 0xb3),
	"light_blue": rgb(0x3a, 0xb3, 0xda),
	"yellow":     rgb(0xfe, 0xd8, 0x3d),
	"lime":       rgb(0x80, 0xc7, 0x1f),
	"pink":       rgb(0xf3, 0x8c, 0xaa),
	"gray":       rgb(0x47, 0x4f, 0x52),
	"light_gray": rgb(0x9c, 0x9d, 0x97),
	"cyan":       rgb(0x16, 0x9c, 0x9d),
	"purple":     rgb(0x89, 0x32, 0xb7),
	"blue":       rgb(0x3c, 0x44, 0xaa),
	"brown":      rgb(0x83, 0x54, 0x32),
	"green":      rgb(0x5d, 0x7c, 0x15),
	"red":        rgb(0xb0, 0x2e, 0x26),
	"black":      rgb(0x1d, 0x1c, 0x21),
}

var andesiteCasingColor = rgb(0x7a, 0x5c, 0x3a)
var sailColor = rgb(0xf5, 0xf0, 0xe0) // cream/off-white for sails

func rgb(r, g, b uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// blockColor determines the display color for a block based on its type and materials.
func blockColor(b generator.Block, mat generator.MaterialConfig) color.RGBA {
	switch b.Type {
	case generator.BlockPlank:
		wood := mat.WoodType
		if wood == "" {
			wood = "oak"
		}
		if w, ok := woodPalette[wood]; ok {
			return w.plank
		}
		return woodPalette["oak"].plank

	case generator.BlockLog:
		if mat.FrameMaterial == "andesite_casing" {
			return andesiteCasingColor
		}
		wood := mat.WoodType
		if wood == "" {
			wood = "oak"
		}
		if w, ok := woodPalette[wood]; ok {
			return w.log
		}
		return woodPalette["oak"].log

	case generator.BlockWool:
		c := mat.EnvelopeColor
		if c == "" {
			c = mat.BladeColor
		}
		if c == "" {
			c = "white"
		}
		if col, ok := woolPalette[c]; ok {
			return col
		}
		return woolPalette["white"]

	case generator.BlockSail:
		return sailColor

	case generator.BlockSlabBot, generator.BlockSlabTop:
		wood := mat.WoodType
		if wood == "" {
			wood = "oak"
		}
		if w, ok := woodPalette[wood]; ok {
			return w.plank
		}
		return woodPalette["oak"].plank

	case generator.BlockStair:
		wood := mat.WoodType
		if wood == "" {
			wood = "oak"
		}
		if w, ok := woodPalette[wood]; ok {
			return w.plank
		}
		return woodPalette["oak"].plank

	case generator.BlockFence:
		wood := mat.WoodType
		if wood == "" {
			wood = "oak"
		}
		if w, ok := woodPalette[wood]; ok {
			return w.plank
		}
		return woodPalette["oak"].plank

	case generator.BlockTrapdoor:
		wood := mat.WoodType
		if wood == "" {
			wood = "oak"
		}
		if w, ok := woodPalette[wood]; ok {
			return w.plank
		}
		return woodPalette["oak"].plank

	default:
		return rgb(0x80, 0x80, 0x80)
	}
}
