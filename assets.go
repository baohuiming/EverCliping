package main

import (
	_ "embed"
)

//go:embed assets/icon.ico
var IconData []byte

//go:embed assets/guide.html
var GuideTemplate string
