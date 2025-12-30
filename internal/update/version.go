package update

import (
	"runtime/debug"
	"strings"
)

// Version information - injected at build time via ldflags:
// go build -ldflags "-X github.com/galaxy-io/tempo/internal/update.Version=1.2.3 -X github.com/galaxy-io/tempo/internal/update.Commit=abc123 -X github.com/galaxy-io/tempo/internal/update.BuildDate=2024-01-01"
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func init() {
	// If version wasn't set via ldflags, try to get it from Go's build info
	// This works when installed via: go install github.com/galaxy-io/tempo/cmd/tempo@latest
	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			// Module version (e.g., "v0.0.6" or "(devel)")
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				Version = info.Main.Version
			}

			// Get VCS info for commit and time
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					if len(setting.Value) >= 7 {
						Commit = setting.Value[:7]
					} else {
						Commit = setting.Value
					}
				case "vcs.time":
					BuildDate = setting.Value
				case "vcs.modified":
					if setting.Value == "true" && !strings.Contains(Version, "dirty") {
						Version += "-dirty"
					}
				}
			}
		}
	}
}
