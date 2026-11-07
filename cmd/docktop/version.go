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
		}
	}

	fmt.Printf("docktop %s\n", version)
	if commit != "none" && commit != "" {
		fmt.Printf("  Commit:   %s\n", commit)
	}
	if date != "unknown" && date != "" {
		fmt.Printf("  Built at: %s\n", date)
	}
	os.Exit(0)
}
