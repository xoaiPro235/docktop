package main

import (
	"flag"
	"fmt"
	"os"
)

type cliOptions struct {
	host string
}

func parseFlags() cliOptions {
	var opts cliOptions
	var showVersion bool

	flag.BoolVar(&showVersion, "v", false, "Show version and exit")
	flag.BoolVar(&showVersion, "version", false, "Show version and exit")

	flag.StringVar(&opts.host, "H", "", "Docker daemon host to connect to")
	flag.StringVar(&opts.host, "host", "", "Alias for -H")

	flag.Usage = printUsage
	flag.Parse()

	if showVersion {
		printVersion()
	}

	return opts
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "docktop - A fast, modern terminal UI for Docker\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  docktop [flags]\n\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	fmt.Fprintf(os.Stderr, "  -H, --host <endpoint>   Docker daemon host to connect to (unix:///..., ssh://...)\n")
	fmt.Fprintf(os.Stderr, "  -v, --version           Show version and exit\n")
	fmt.Fprintf(os.Stderr, "  -h, --help              Show help information\n\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  $ docktop\n")
	fmt.Fprintf(os.Stderr, "  $ docktop -H unix:///var/run/docker.sock\n")
	fmt.Fprintf(os.Stderr, "  $ docktop -H ssh://user@remote-host\n")
}
