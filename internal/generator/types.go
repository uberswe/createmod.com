package generator

const CurrentVersion = 1

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

func isValidWoodType(s string) bool {
	for _, v := range ValidWoodTypes {
		if s == v {
			return true
		}
	}
	return false
}

func isValidWoolColor(s string) bool {
	for _, v := range ValidWoolColors {
		if s == v {
			return true
		}
	}
	return false
}
