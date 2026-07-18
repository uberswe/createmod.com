package schematic

import (
	"strings"

	"github.com/Tnze/go-mc/nbt"
)

// IsCopycat reports whether a block is a Create copycat (create:copycat_*)
// or a Copycats+ variant (mod id "copycats"). Copycats wrap another block,
// stored as a block state compound in the block entity's Material tag.
func IsCopycat(name string) bool {
	return strings.HasPrefix(name, "create:copycat_") || strings.HasPrefix(name, "copycats:")
}

// CopycatMaterialName reads the wrapped block id from a copycat block
// entity's Material tag ({Name, Properties}). Returns "" when the tag is
// missing, malformed, or air (no material applied yet).
func CopycatMaterialName(raw nbt.RawMessage) string {
	fields, err := compoundFields(raw)
	if err != nil {
		return ""
	}
	mat, ok := fields["Material"]
	if !ok || mat.Type != nbt.TagCompound {
		return ""
	}
	mf, err := compoundFields(mat)
	if err != nil {
		return ""
	}
	rfm := rawFieldsMap{}
	for k, v := range mf {
		rfm[k] = rawField(v)
	}
	name, _ := strField(rfm, "Name")
	if name == "" || name == "minecraft:air" || !strings.Contains(name, ":") {
		return ""
	}
	return name
}
