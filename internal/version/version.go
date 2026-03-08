package version

import "strings"

// Injected at build time via ldflags. See Makefile / .goreleaser.yml.
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// ImageTag returns the Docker image tag corresponding to this CLI version.
// Release builds (e.g. "v0.1.0") use the version directly; dev builds fall
// back to "latest". Git metadata suffixes like "-dirty" or "-3-gabcdef" are
// stripped so the tag always matches a real image.
func ImageTag() string {
	if Version == "dev" {
		return "latest"
	}
	tag := Version
	tag = strings.TrimSuffix(tag, "-dirty")
	// Strip git describe distance suffix (e.g. "v0.1.0-3-gabcdef" → "v0.1.0")
	if idx := strings.Index(tag, "-"); idx > 0 {
		if rest := tag[idx+1:]; len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
			tag = tag[:idx]
		}
	}
	return tag
}
