package version

import "regexp"

// Injected at build time via ldflags. See Makefile / .goreleaser.yml.
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var releaseTagPattern = regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)

// RecommendedOpenClawVersion is the OpenClaw version that has been tested
// with this release of ClawFleet. Updated with each ClawFleet release.
const RecommendedOpenClawVersion = "2026.4.11"

// ImageTag returns the Docker image tag corresponding to this CLI version.
// Only exact release tags (e.g. "v0.1.0") map to release images. Local git
// describe builds such as "v0.1.0-44-gabcdef" or dirty builds fall back to
// "latest" so development binaries do not accidentally target an old release
// image.
func ImageTag() string {
	if releaseTagPattern.MatchString(Version) {
		return Version
	}
	return "latest"
}
