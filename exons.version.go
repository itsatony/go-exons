package exons

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed versions.yaml
var versionsYAML []byte

// versionInfo holds the parsed content of versions.yaml.
type versionInfo struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
	GoVersion   string `yaml:"go_version"`
	Repository  string `yaml:"repository"`
	License     string `yaml:"license"`
	Website     string `yaml:"website"`
}

// Version is the current library version, loaded from versions.yaml at compile time.
var Version string

func init() {
	var info versionInfo
	if err := yaml.Unmarshal(versionsYAML, &info); err != nil {
		Version = "0.0.0-unknown"
		return
	}
	Version = info.Version
}
