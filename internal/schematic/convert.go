package schematic

import (
	"fmt"

	"github.com/Tnze/go-mc/nbt"
)

// Format identifies a schematic file format.
type Format string

const (
	FormatStructure Format = "nbt"       // vanilla structure / Create .nbt
	FormatSponge    Format = "schem"     // Sponge v2/v3 .schem
	FormatLitematic Format = "litematic" // Litematica .litematic
	FormatLegacy    Format = "schematic" // MCEdit/WorldEdit legacy .schematic (read: phase C2)
	FormatSable     Format = "sable"     // Sable Blueprint v1 (detect-only for now)
	FormatBlueprint Format = "blueprint" // Structurize / MineColonies .blueprint
	FormatBG        Format = "bg"        // Building Gadgets template (.json / .txt)
	FormatUnknown   Format = ""
)

// formatProbe decodes just enough root fields to identify a format. Content
// is sniffed — extensions are never trusted.
type formatProbe struct {
	// structure
	Size    nbt.RawMessage `nbt:"size"`
	Palette nbt.RawMessage `nbt:"palette"`
	Blocks  nbt.RawMessage `nbt:"blocks"`
	// sponge v1/v2
	Version   nbt.RawMessage `nbt:"Version"`
	BlockData nbt.RawMessage `nbt:"BlockData"`
	PaletteS  nbt.RawMessage `nbt:"Palette"`
	// sponge v3 nesting
	Schematic nbt.RawMessage `nbt:"Schematic"`
	// litematic
	Regions              nbt.RawMessage `nbt:"Regions"`
	MinecraftDataVersion nbt.RawMessage `nbt:"MinecraftDataVersion"`
	// legacy MCEdit
	Materials nbt.RawMessage `nbt:"Materials"`
	Data      nbt.RawMessage `nbt:"Data"`
	BlocksArr nbt.RawMessage `nbt:"Blocks"`
	// Sable blueprint v1 (lowercase root tags)
	SableVersion   nbt.RawMessage `nbt:"version"`
	SableSubLevels nbt.RawMessage `nbt:"sub_levels"`
	// Structurize blueprint (shares the "palette" probe field above)
	BpSizeX nbt.RawMessage `nbt:"size_x"`
}

// Detect identifies the format of a schematic file by content.
func Detect(data []byte) (Format, error) {
	// Building Gadgets templates are JSON text, not NBT.
	if looksLikeJSON(data) {
		return FormatBG, nil
	}
	raw, err := decompress(data)
	if err != nil {
		return FormatUnknown, err
	}
	var p formatProbe
	if err := nbt.Unmarshal(raw, &p); err != nil {
		return FormatUnknown, fmt.Errorf("schematic: not an NBT file: %w", err)
	}
	has := func(r nbt.RawMessage) bool { return r.Type != nbt.TagEnd && len(r.Data) > 0 }

	switch {
	// Sable Blueprint v1 sniff: lowercase version + sub_levels root tags
	// distinguish it from vanilla structure NBT (checked first — a Sable
	// file has no size/palette/Regions roots, but be explicit anyway).
	case has(p.SableVersion) && has(p.SableSubLevels):
		return FormatSable, nil
	// Structurize blueprint: size_x + palette (+ byte version, but the
	// size_x short is the distinctive root tag)
	case has(p.BpSizeX) && has(p.Palette):
		return FormatBlueprint, nil
	case has(p.Size) && (has(p.Palette) || has(p.Blocks)):
		return FormatStructure, nil
	case has(p.Regions) && has(p.MinecraftDataVersion):
		return FormatLitematic, nil
	case has(p.Schematic) || (has(p.Version) && (has(p.PaletteS) || has(p.BlockData))):
		return FormatSponge, nil
	case has(p.Materials) && has(p.BlocksArr) && has(p.Data):
		return FormatLegacy, nil
	default:
		return FormatUnknown, fmt.Errorf("schematic: unrecognized schematic format")
	}
}

// Read parses data in the given format (use Detect first) into the model.
func Read(data []byte, f Format) (*Schematic, error) {
	switch f {
	case FormatStructure:
		return ReadStructureNBT(data)
	case FormatSponge:
		return ReadSponge(data)
	case FormatLitematic:
		return ReadLitematic(data)
	case FormatLegacy:
		return ReadLegacy(data)
	case FormatSable:
		return ReadSable(data)
	case FormatBlueprint:
		return ReadBlueprint(data)
	case FormatBG:
		return ReadBuildingGadgets(data)
	default:
		return nil, fmt.Errorf("schematic: unknown format %q", f)
	}
}

// Write serializes the model in the given format.
func Write(s *Schematic, f Format) ([]byte, error) {
	switch f {
	case FormatStructure:
		return WriteStructureNBT(s)
	case FormatSponge:
		return WriteSponge(s)
	case FormatLitematic:
		return WriteLitematic(s)
	case FormatLegacy:
		out, _, err := WriteLegacy(s)
		return out, err
	case FormatBlueprint:
		return WriteBlueprint(s)
	case FormatBG:
		out, _, err := WriteBuildingGadgets(s)
		return out, err
	case FormatSable:
		return nil, fmt.Errorf("schematic: writing Sable blueprints is not supported while the format is experimental")
	default:
		return nil, fmt.Errorf("schematic: unknown format %q", f)
	}
}

// Warning describes a fidelity loss incurred during conversion.
type Warning struct {
	Message string `json:"message"`
}

// ConvertResult carries the converted bytes plus any lossiness warnings.
type ConvertResult struct {
	Data     []byte
	From     Format
	To       Format
	Warnings []Warning
}

// Convert detects the input format, parses, and re-serializes as target.
// Same-format input is normalized (canonical writer output) rather than
// passed through, so output is always byte-stable and hardened.
func Convert(data []byte, target Format) (*ConvertResult, error) {
	from, err := Detect(data)
	if err != nil {
		return nil, err
	}
	s, err := Read(data, from)
	if err != nil {
		return nil, err
	}
	res := &ConvertResult{From: from, To: target}
	if target == FormatLegacy {
		out, warnings, err := WriteLegacy(s)
		if err != nil {
			return nil, err
		}
		res.Data = out
		res.Warnings = append(res.Warnings, warnings...)
	} else if target == FormatBG {
		out, warnings, err := WriteBuildingGadgets(s)
		if err != nil {
			return nil, err
		}
		res.Data = out
		res.Warnings = append(res.Warnings, warnings...)
	} else {
		out, err := Write(s, target)
		if err != nil {
			return nil, err
		}
		res.Data = out
	}
	for _, note := range s.Meta.LossyNotes {
		res.Warnings = append(res.Warnings, Warning{Message: note})
	}
	if s.DataVersion == 0 && target != FormatStructure {
		res.Warnings = append(res.Warnings, Warning{Message: "source carries no Minecraft DataVersion; the output declares version 0 and may need adjustment"})
	}
	return res, nil
}
