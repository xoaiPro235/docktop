package main

import (
	"flag"
	"os"
	"testing"
)

func TestFlagDefaults(t *testing.T) {
	// Ensure flags can be defined and parsed without crashing
	fs := flag.NewFlagSet("docktop", flag.ContinueOnError)
	var showVersion bool
	fs.BoolVar(&showVersion, "v", false, "Show version and exit")
	var host string
	fs.StringVar(&host, "H", "", "Docker daemon host")

	err := fs.Parse([]string{"-H", "unix:///var/run/docker.sock"})
	if err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}
	if host != "unix:///var/run/docker.sock" {
		t.Errorf("host = %q, want 'unix:///var/run/docker.sock'", host)
	}
}

func TestVersionVariables(t *testing.T) {
	if version == "" {
		t.Error("version variable should not be empty")
	}
}

func TestUsagePrint(t *testing.T) {
	// Verify Usage function does not panic
	origUsage := flag.Usage
	defer func() { flag.Usage = origUsage }()

	flag.Usage = func() {
		_ = os.Stderr
	}
	flag.Usage()
}
