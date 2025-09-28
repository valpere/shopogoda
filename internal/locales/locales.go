package locales

import (
	"embed"
)

// LocalesFS contains embedded locale files
//
//go:embed *.json
var LocalesFS embed.FS
