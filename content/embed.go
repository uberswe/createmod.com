package content

import "embed"

//go:embed news/*.md
var NewsFS embed.FS
