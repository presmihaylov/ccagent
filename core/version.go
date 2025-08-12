package core

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionContent string

// GetVersion returns the current version of ccagent
func GetVersion() string {
	return strings.TrimSpace(versionContent)
}