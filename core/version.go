package core

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionContent string

// GetVersion returns the current version of eksecd
func GetVersion() string {
	return strings.TrimSpace(versionContent)
}
