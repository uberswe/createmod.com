package schematic

// Tier-1 hardening limits. Every reader in this package enforces these; they
// consolidate (and match or tighten) the caps historically applied in
// internal/nbtparser so a hostile file cannot exhaust memory or CPU.
const (
	// MaxDecompressedSize caps NBT payloads after gzip/zlib decompression
	// (decompression-bomb guard).
	MaxDecompressedSize = 100 * 1024 * 1024

	// MaxDimension caps each axis of a schematic's bounding box.
	MaxDimension = 32768

	// MaxVolume caps the total bounding-box volume so Size values that are
	// individually legal cannot multiply into an allocation bomb.
	MaxVolume = 64 * 1024 * 1024

	// MaxPaletteSize caps distinct block states.
	MaxPaletteSize = 65536

	// MaxBlockIDLength caps a block identifier's length.
	MaxBlockIDLength = 256

	// MaxBlockEntities caps the number of block entities.
	MaxBlockEntities = 1 << 20

	// MaxEntities caps the number of entities.
	MaxEntities = 1 << 16
)
