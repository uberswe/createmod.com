package pages

import (
	"strings"
	"testing"
)

func Test_sanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantExt  string
		checkFn  func(t *testing.T, result string)
	}{
		{
			name:    "ascii filename unchanged",
			input:   "my-schematic.nbt",
			wantExt: ".nbt",
			checkFn: func(t *testing.T, result string) {
				if result != "my-schematic.nbt" {
					t.Errorf("expected my-schematic.nbt, got %s", result)
				}
			},
		},
		{
			name:    "spaces replaced with underscores",
			input:   "my schematic file.nbt",
			wantExt: ".nbt",
			checkFn: func(t *testing.T, result string) {
				if strings.Contains(result, " ") {
					t.Errorf("result should not contain spaces: %s", result)
				}
				if result != "my_schematic_file.nbt" {
					t.Errorf("expected my_schematic_file.nbt, got %s", result)
				}
			},
		},
		{
			name:    "japanese characters stripped",
			input:   "スクリーンショット 2026-03-09 041703.webp",
			wantExt: ".webp",
			checkFn: func(t *testing.T, result string) {
				if strings.Contains(result, "スクリーン") {
					t.Errorf("result should not contain Japanese characters: %s", result)
				}
				if !strings.HasSuffix(result, ".webp") {
					t.Errorf("result should end with .webp: %s", result)
				}
				// Should retain the numeric portion
				if !strings.Contains(result, "2026-03-09") {
					t.Errorf("result should contain the date portion: %s", result)
				}
			},
		},
		{
			name:    "purely non-ascii generates random name",
			input:   "日本語.nbt",
			wantExt: ".nbt",
			checkFn: func(t *testing.T, result string) {
				if !strings.HasSuffix(result, ".nbt") {
					t.Errorf("result should end with .nbt: %s", result)
				}
				base := strings.TrimSuffix(result, ".nbt")
				if len(base) == 0 {
					t.Error("base name should not be empty")
				}
			},
		},
		{
			name:    "mixed unicode and ascii",
			input:   "café_design.png",
			wantExt: ".png",
			checkFn: func(t *testing.T, result string) {
				if !strings.HasSuffix(result, ".png") {
					t.Errorf("result should end with .png: %s", result)
				}
				if strings.Contains(result, "é") {
					t.Errorf("result should not contain non-ASCII: %s", result)
				}
			},
		},
		{
			name:    "preserves extension case as lowercase",
			input:   "TEST.NBT",
			wantExt: ".nbt",
			checkFn: func(t *testing.T, result string) {
				if !strings.HasSuffix(result, ".nbt") {
					t.Errorf("result should end with .nbt: %s", result)
				}
			},
		},
		{
			name:    "special characters stripped",
			input:   "my file (copy) [2].nbt",
			wantExt: ".nbt",
			checkFn: func(t *testing.T, result string) {
				if strings.ContainsAny(result, "()[] ") {
					t.Errorf("result should not contain special characters: %s", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)

			// Check extension
			if !strings.HasSuffix(result, tt.wantExt) {
				t.Errorf("sanitizeFilename(%q) = %q, want extension %q", tt.input, result, tt.wantExt)
			}

			// Check no spaces
			if strings.Contains(result, " ") {
				t.Errorf("sanitizeFilename(%q) = %q, should not contain spaces", tt.input, result)
			}

			// Run custom check
			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}
