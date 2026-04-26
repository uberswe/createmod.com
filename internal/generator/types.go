package generator

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
)

type GenerateResult struct {
	Blocks []Block `json:"blocks"`
	SizeX  int     `json:"sizeX"`
	SizeY  int     `json:"sizeY"`
	SizeZ  int     `json:"sizeZ"`
}
