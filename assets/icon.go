// Package assets owns the Windows icon embedded in the application and tray.
package assets

import _ "embed"

// Icon is generated from assets/generate.go and also used by Winres.
//
//go:embed app.ico
var Icon []byte
