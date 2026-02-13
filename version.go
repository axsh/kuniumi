package kuniumi

import (
	"fmt"
	"runtime/debug"
)

// frameworkVersionString returns a string describing the kuniumi framework version
// and the application's VCS commit information.
// Format: "based on kuniumi <version> <commit>"
// Fallback: "based on kuniumi dev" if build info is unavailable.
func frameworkVersionString() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "based on kuniumi dev"
	}

	// Get kuniumi module version from dependencies
	fwVersion := "dev"
	for _, dep := range info.Deps {
		if dep.Path == "github.com/axsh/kuniumi" {
			fwVersion = dep.Version
			break
		}
	}

	// If this IS the kuniumi module itself (during development),
	// use the main module version
	if fwVersion == "dev" && info.Main.Path == "github.com/axsh/kuniumi" {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			fwVersion = info.Main.Version
		}
	}

	// Get VCS revision and modified status from build settings
	var revision string
	var modified bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) > 7 {
				revision = s.Value[:7]
			} else {
				revision = s.Value
			}
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}

	// Build the version string
	result := fmt.Sprintf("based on kuniumi %s", fwVersion)
	if revision != "" {
		result += " " + revision
		if modified {
			result += "-dirty"
		}
	}

	return result
}
