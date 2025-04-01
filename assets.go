package main

import (
	_ "embed"
)

//go:embed assets/icon.ico
var IconData []byte

//go:embed assets/guide.html
var GuideTemplate string

// Please run generate.bat to update this file
//
//go:embed assets/shortcuts.png
var ShortcutsData []byte

//go:embed assets/shortcuts.url
var ShortcutsURL string
