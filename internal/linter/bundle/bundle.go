package bundle

import (
	"embed"
)

//go:embed *
var BundleFS embed.FS
