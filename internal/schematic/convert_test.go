package schematic

import (
	"reflect"
	"testing"
)

func Test_Detect(t *testing.T) {
	structure := generatorFixture(t)
	s, _ := ReadStructureNBT(structure)
	schem, err := WriteSponge(s)
	if err != nil {
		t.Fatal(err)
	}
	lit, err := WriteLitematic(s)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]struct {
		data []byte
		want Format
	}{
		"structure": {structure, FormatStructure},
		"sponge":    {schem, FormatSponge},
		"litematic": {lit, FormatLitematic},
	}
	for name, c := range cases {
		got, err := Detect(c.data)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: detected %q, want %q", name, got, c.want)
		}
	}

	if _, err := Detect([]byte("garbage")); err == nil {
		t.Errorf("garbage detected as a format")
	}
}

// Full conversion matrix: every supported source format converts to every
// supported target and the block content survives.
func Test_Convert_Matrix(t *testing.T) {
	src, err := ReadStructureNBT(handmadeFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	formats := []Format{FormatStructure, FormatSponge, FormatLitematic}
	inputs := map[Format][]byte{}
	for _, f := range formats {
		data, err := Write(src, f)
		if err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
		inputs[f] = data
	}

	wantMaterials := src.Materials()
	for _, from := range formats {
		for _, to := range formats {
			res, err := Convert(inputs[from], to)
			if err != nil {
				t.Errorf("%s -> %s: %v", from, to, err)
				continue
			}
			if res.From != from {
				t.Errorf("%s -> %s: detected source as %s", from, to, res.From)
			}
			back, err := Read(res.Data, to)
			if err != nil {
				t.Errorf("%s -> %s: output unreadable: %v", from, to, err)
				continue
			}
			if back.Size != src.Size {
				t.Errorf("%s -> %s: size %v -> %v", from, to, src.Size, back.Size)
			}
			if back.DataVersion != src.DataVersion {
				t.Errorf("%s -> %s: DataVersion %d -> %d", from, to, src.DataVersion, back.DataVersion)
			}
			if !reflect.DeepEqual(back.Materials(), wantMaterials) {
				t.Errorf("%s -> %s: materials changed", from, to)
			}
			if len(back.BlockEntities) != len(src.BlockEntities) {
				t.Errorf("%s -> %s: block entities %d -> %d", from, to, len(src.BlockEntities), len(back.BlockEntities))
			}
		}
	}
}

func Test_Convert_UnsupportedTargets(t *testing.T) {
	data := generatorFixture(t)
	// Legacy is a supported (lossy) target and must always carry warnings.
	res, err := Convert(data, FormatLegacy)
	if err != nil {
		t.Errorf("legacy write: %v", err)
	} else if len(res.Warnings) == 0 {
		t.Errorf("legacy conversion must carry lossiness warnings")
	}
	if _, err := Convert(data, FormatSable); err == nil {
		t.Errorf("sable write should be unsupported")
	}
}

func FuzzDetectAndRead(f *testing.F) {
	t := &testing.T{}
	src, _ := ReadStructureNBT(handmadeFixture(t))
	for _, format := range []Format{FormatStructure, FormatSponge, FormatLitematic, FormatLegacy} {
		if data, err := Write(src, format); err == nil {
			f.Add(data)
		}
	}
	f.Add(sableFixture(t))
	f.Fuzz(func(t *testing.T, data []byte) {
		format, err := Detect(data)
		if err != nil {
			return
		}
		s, err := Read(data, format)
		if err != nil {
			return
		}
		if err := s.Validate(); err != nil {
			t.Fatalf("accepted %s fails validation: %v", format, err)
		}
	})
}
