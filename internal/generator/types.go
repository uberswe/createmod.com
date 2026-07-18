package generator

// CurrentVersion is the version stamped on new generations and share hashes.
// The version selects both the hash float encoding and the hull algorithm:
//
//	1    legacy hash encoding (raw floats)
//	2    hash floats stored x100; hull algorithm v1
//	3    hull algorithm v2 (lofted sections, profile curves); same encoding as 2
//
// Versions <= 2 must reproduce byte-identical results forever: existing share
// links embed the version and regenerate on every visit. Never modify
// generateHullV1 or the v1 preset semantics.
const CurrentVersion = 3

// HullV2MinVersion is the first version that uses the v2 hull algorithm.
const HullV2MinVersion = 3

type Block struct {
	X     int               `json:"x"`
	Y     int               `json:"y"`
	Z     int               `json:"z"`
	Type  int               `json:"type"`
	Props map[string]string `json:"props,omitempty"`
}

const (
	BlockAir      = 0
	BlockPlank    = 1
	BlockSlabBot  = 2
	BlockSlabTop  = 3
	BlockStair    = 4
	BlockFence    = 5
	BlockTrapdoor = 6
	BlockWool     = 7
	BlockLog      = 8
	BlockSail     = 9
)

type MaterialConfig struct {
	WoodType         string `json:"woodType,omitempty"`
	EnvelopeMaterial string `json:"envelopeMaterial,omitempty"`
	EnvelopeColor    string `json:"envelopeColor,omitempty"`
	BladeMaterial    string `json:"bladeMaterial,omitempty"`
	BladeColor       string `json:"bladeColor,omitempty"`
	FrameMaterial    string `json:"frameMaterial,omitempty"`
	SailFacing       string `json:"sailFacing,omitempty"`
	Orientation      string `json:"orientation,omitempty"`
}

type GenerateResult struct {
	Blocks    []Block        `json:"blocks"`
	SizeX     int            `json:"sizeX"`
	SizeY     int            `json:"sizeY"`
	SizeZ     int            `json:"sizeZ"`
	Materials MaterialConfig `json:"materials"`
}

var ValidWoodTypes = []string{
	"oak", "spruce", "birch", "dark_oak", "jungle", "acacia", "cherry", "crimson", "warped",
}

var ValidWoolColors = []string{
	"white", "orange", "magenta", "light_blue", "yellow", "lime", "pink",
	"gray", "light_gray", "cyan", "purple", "blue", "brown", "green", "red", "black",
}

var validWoodTypes = make(map[string]bool)
var validWoolColors = make(map[string]bool)

func init() {
	for _, v := range ValidWoodTypes {
		validWoodTypes[v] = true
	}
	for _, v := range ValidWoolColors {
		validWoolColors[v] = true
	}
}

func isValidWoodType(s string) bool  { return validWoodTypes[s] }
func isValidWoolColor(s string) bool { return validWoolColors[s] }
