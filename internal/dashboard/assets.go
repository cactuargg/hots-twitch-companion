package dashboard

import (
	"embed"
)

// WebFiles holds the application HTML/CSS/JS, keeping the same "web/" subtree
// expected by the existing HTTP and Wails handlers.
//
//go:embed web/* web/assets/*
var WebFiles embed.FS
