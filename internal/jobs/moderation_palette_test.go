package jobs

import (
	"strings"
	"testing"

	"createmod/internal/nbtparser"
)

func TestSummarizeBlockPalette(t *testing.T) {
	if got := summarizeBlockPalette(nil); got != "" {
		t.Errorf("empty palette = %q, want empty", got)
	}

	mats := []nbtparser.Material{
		{BlockID: "minecraft:andesite", Count: 80},
		{BlockID: "create:cogwheel", Count: 120},
		{BlockID: "create:shaft", Count: 40},
	}
	got := summarizeBlockPalette(mats)

	// Header reports distinct types and total building blocks.
	if !strings.Contains(got, "3 distinct block types") {
		t.Errorf("missing distinct-types header in %q", got)
	}
	if !strings.Contains(got, "240 building blocks total") {
		t.Errorf("missing total-count header in %q", got)
	}
	// Most common block must be listed first.
	cog := strings.Index(got, "create:cogwheel")
	and := strings.Index(got, "minecraft:andesite")
	shaft := strings.Index(got, "create:shaft")
	if !(cog >= 0 && cog < and && and < shaft) {
		t.Errorf("blocks not ordered by count desc: %q", got)
	}
}

func TestSummarizeBlockPalette_CapsAndDeterministic(t *testing.T) {
	mats := make([]nbtparser.Material, 0, moderationPaletteTopN+5)
	for i := 0; i < moderationPaletteTopN+5; i++ {
		// Descending counts so ordering is unambiguous.
		mats = append(mats, nbtparser.Material{BlockID: "minecraft:block_" + string(rune('a'+i%26)) + itoa(i), Count: 1000 - i})
	}
	got := summarizeBlockPalette(mats)
	lines := strings.Count(got, "\n- ")
	// Header line + at most topN block lines + one "and N more" line.
	if strings.Count(got, "- ") > moderationPaletteTopN+1 {
		t.Errorf("listed more than topN(%d)+overflow rows: %d bullets", moderationPaletteTopN, strings.Count(got, "- "))
	}
	if !strings.Contains(got, "and 5 more block types") {
		t.Errorf("missing overflow line in %q", got)
	}
	_ = lines
	// Deterministic: same input, same output.
	if got != summarizeBlockPalette(mats) {
		t.Error("summary is not deterministic")
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
