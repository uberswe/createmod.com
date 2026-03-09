package modmeta

import (
	_ "embed"
	"strings"

	"github.com/go-ego/gse"
)

//go:embed dict/count_1w.txt
var englishDict string

var seg gse.Segmenter

func init() {
	seg.AlphaNum = true
	// Norvig corpus is tab-separated (word\tfreq); gse expects space-separated
	seg.LoadDictStr(strings.ReplaceAll(englishDict, "\t", " "))
}
