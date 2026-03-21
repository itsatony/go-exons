package exons

import (
	_ "embed"

	goversion "github.com/itsatony/go-version"
)

//go:embed versions.yaml
var versionsYAML []byte

// Version is the current library version, loaded from the embedded versions.yaml
// via go-version. This is the single source of truth — edit versions.yaml to bump.
var Version string

func init() {
	if err := goversion.Initialize(goversion.WithEmbedded(versionsYAML)); err != nil {
		Version = "0.0.0-unknown"
		return
	}
	Version = goversion.MustGet().Project.Version
}

// VersionInfo returns the full go-version Info including project, git, and build metadata.
// Returns nil if version initialization failed.
func VersionInfo() *goversion.Info {
	info, err := goversion.Get()
	if err != nil {
		return nil
	}
	return info
}
