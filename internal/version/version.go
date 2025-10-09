package version

import (
	"fmt"
	"runtime"
)

// Version information - set at build time via ldflags
var (
	// Version is the semantic version of the application
	Version = "0.1.1"

	// GitCommit is the git commit SHA
	GitCommit = "unknown"

	// BuildTime is the build timestamp
	BuildTime = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = runtime.Version()
)

// Info represents complete version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

// GetInfo returns version information
func GetInfo() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
	}
}

// String returns formatted version information
func (i Info) String() string {
	return fmt.Sprintf("ShoPogoda v%s\nCommit: %s\nBuilt: %s\nGo: %s",
		i.Version, i.GitCommit, i.BuildTime, i.GoVersion)
}

// Short returns short version string
func (i Info) Short() string {
	commit := i.GitCommit
	if len(commit) > 7 {
		commit = commit[:7]
	}
	return fmt.Sprintf("v%s (%s)", i.Version, commit)
}
