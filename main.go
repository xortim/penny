package main

import (
	_ "embed"

	"github.com/xortim/penny/cmd"
)

//go:embed CHANGELOG.md
var changelogRaw string

func main() {
	cmd.ChangelogRaw = changelogRaw
	// Add your plugin handlers to cmd/server.go
	cmd.Execute()
}
