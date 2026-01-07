package main

import "embed"

//go:embed seeds/*.json
var seedFiles embed.FS
