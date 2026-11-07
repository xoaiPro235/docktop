package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
)

// Injected by GoReleaser ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func printVersion() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" && !strings.HasPrefix(info.Main.Version, "v0.0.0-") {
				version = info.Main.Version
			}
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					commit = setting.Value
					if len(commit) > 7 {
						commit = commit[:7]
					}
				case "vcs.time":
					date = setting.Value
				}
			}
		}
	}

	fmt.Printf("docktop %s\n\n", version)
	fmt.Printf("%16s %s\n", "Commit:", commit)
	fmt.Printf("%16s %s\n", "Built at:", date)
	os.Exit(0)
}
