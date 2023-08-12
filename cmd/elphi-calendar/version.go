package main

import (
	"fmt"
	"os"
	"runtime/debug"
)

var (
	// The Version string as set in the git tag.
	// The format is expected to be v1.2.3 or v1.2.3-pre where pre is a prerelease identifier.
	// Except for the "v" prefix the format should be a semantic version without build info.
	// Set via ldflags.
	Version string
)

// printVersion prints the current elphi-calendar version.
func printVersion() {
	info := getVersionInfo()
	if info == nil {
		_, _ = fmt.Fprintln(os.Stderr, "No version info provided during build.")
		os.Exit(1)
	}

	if len(info.GitCommit) > 12 {
		info.GitCommit = info.GitCommit[:12]
	}
	dirty := ""
	if info.GitTreeState == "dirty" {
		dirty = ".dirty"
	}

	if info.GitVersion == "" {
		fmt.Printf("elphi-calendar development build (built with %s)\n", info.GoVersion)
		return
	}
	fmt.Printf("elphi-calendar %s+git.%s%s (built with %s)\n", info.GitVersion, info.GitCommit, dirty, info.GoVersion)
}

// versionInfo contains the data for the version command output.
type versionInfo struct {
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	GitTreeState string `json:"gitTreeState"`
	GoVersion    string `json:"goVersion"`
}

// getVersionInfo generates a versionInfo value from the version info provided at build time.
// If no build time version info was provided, nil is returned.
func getVersionInfo() *versionInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}
	v := &versionInfo{
		GoVersion:  info.GoVersion,
		GitVersion: Version,
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			v.GitCommit = setting.Value
		case "vcs.modified":
			if setting.Value == "true" {
				v.GitTreeState = "dirty"
			} else {
				v.GitTreeState = "clean"
			}
		}
	}
	return v
}
