package pages

import (
	"fmt"
	"path/filepath"
	"strings"

	"createmod/internal/schematic"
)

// uploadableSchematicExts are the file extensions accepted by schematic
// uploads. Everything that is not Create/vanilla structure NBT is converted
// to it at upload time, so the rest of the pipeline (validation, stats,
// dedup, storage, mod downloads) keeps operating on .nbt only.
var uploadableSchematicExts = []string{
	".nbt", ".schem", ".litematic", ".schematic", ".blueprint", ".txt", ".json",
}

// UploadAcceptAttr is the <input accept> value for schematic upload forms.
const UploadAcceptAttr = ".nbt,.schem,.litematic,.schematic,.blueprint,.txt,.json"

func isUploadableSchematicName(filename string) bool {
	lower := strings.ToLower(filename)
	for _, ext := range uploadableSchematicExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// normalizeUploadToNBT sniffs the uploaded schematic's format by content and
// converts it to Create/vanilla structure NBT when needed. Returns the
// (possibly converted) bytes, the filename to store (extension rewritten to
// .nbt), the detected source format slug, and human-readable conversion
// warnings. Structure-NBT input passes through untouched.
func normalizeUploadToNBT(filename string, data []byte) (out []byte, outName string, sourceFormat string, warnings []string, err error) {
	format, err := schematic.Detect(data)
	if err != nil {
		return nil, "", "", nil, fmt.Errorf("unrecognized schematic format: %s", convertUserError(err))
	}
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	if format == schematic.FormatStructure {
		return data, base + ".nbt", string(format), nil, nil
	}
	res, err := schematic.Convert(data, schematic.FormatStructure)
	if err != nil {
		return nil, "", "", nil, fmt.Errorf("could not convert %s to Create .nbt: %s", format, convertUserError(err))
	}
	for _, w := range res.Warnings {
		warnings = append(warnings, w.Message)
	}
	return res.Data, base + ".nbt", string(format), warnings, nil
}
