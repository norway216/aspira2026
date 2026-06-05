package main

import "embed"

//go:embed web/static
var StaticFS embed.FS

//go:embed web/templates/index.html
var IndexHTML string
