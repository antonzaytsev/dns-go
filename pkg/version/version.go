// Package version provides build and version information for the DNS server.
package version

import (
	"fmt"
	"runtime"
	"time"
)

// Build information. These values are injected at build time using ldflags.
var (
	// Version is the semantic version of the build.
	Version = "dev"

	// GitCommit is the git commit hash of the build.
	GitCommit = "unknown"

	// BuildDate is the date when the binary was built.
	BuildDate = "unknown"

	// GoVersion is the version of Go used to build the binary.
	GoVersion = runtime.Version()
)

// Info contains all version and build information.
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get returns the version information.
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a human-readable version string.
func (i Info) String() string {
	if i.GitCommit == "unknown" {
		return fmt.Sprintf("dns-server version %s %s %s",
			i.Version, i.GoVersion, i.Platform)
	}

	buildDate := i.BuildDate
	if t, err := time.Parse(time.RFC3339, i.BuildDate); err == nil {
		buildDate = t.Format("2006-01-02T15:04:05Z")
	}

	return fmt.Sprintf("dns-server version %s (commit %s, built %s) %s %s",
		i.Version, i.GitCommit[:8], buildDate, i.GoVersion, i.Platform)
}

// Short returns a short version string.
func (i Info) Short() string {
	if i.GitCommit == "unknown" {
		return i.Version
	}
	return fmt.Sprintf("%s-%s", i.Version, i.GitCommit[:8])
}
