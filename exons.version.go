package exons

import (
	_ "embed"

	goversion "github.com/itsatony/go-version"
)

//go:embed versions.yaml
var versionsYAML []byte

// versionInfo holds the library's own version instance (not the global singleton).
// Libraries must use New() instead of Initialize() to avoid claiming the singleton
// that belongs to the application's main().
var versionInfo *goversion.Info

// Version is the current library version, loaded from the embedded versions.yaml
// via go-version. This is the single source of truth — edit versions.yaml to bump.
var Version string

func init() {
	info, err := goversion.New(goversion.WithEmbedded(versionsYAML))
	if err != nil {
		Version = "0.0.0-unknown"
		return
	}
	versionInfo = info
	Version = info.Project.Version
}

// VersionInfo returns the full go-version Info including project, git, and build metadata.
// Returns nil if version initialization failed.
func VersionInfo() *goversion.Info {
	return versionInfo
}
