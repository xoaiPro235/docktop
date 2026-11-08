package main

import (
	"flag"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		host string
	}{
		{name: "defaults"},
		{name: "short host flag", args: []string{"-H", "unix:///var/run/docker.sock"}, host: "unix:///var/run/docker.sock"},
		{name: "long host flag", args: []string{"--host", "ssh://user@host"}, host: "ssh://user@host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := parseFlagsForTest(t, tt.args...)
			if opts.host != tt.host {
				t.Errorf("host = %q, want %q", opts.host, tt.host)
			}
		})
	}
}

func TestVersionVariables(t *testing.T) {
	if version == "" || commit == "" || date == "" {
		t.Errorf("version metadata should not be empty: version=%q commit=%q date=%q", version, commit, date)
	}
}

func TestUsagePrint(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	originalStderr := os.Stderr
	os.Stderr = write
	printUsage()
	os.Stderr = originalStderr
	if err := write.Close(); err != nil {
		t.Fatal(err)
	}

	output, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	if err := read.Close(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"docktop - A fast, modern terminal UI for Docker", "-H, --host <endpoint>", "-v, --version", "-h, --help"} {
		if !strings.Contains(string(output), want) {
			t.Errorf("usage output %q does not contain %q", output, want)
		}
	}
}

func TestVersionFlagExitsBeforeDocker(t *testing.T) {
	if os.Getenv("DOCKTOP_VERSION_TEST") == "1" {
		parseFlagsForTest(t, os.Getenv("DOCKTOP_VERSION_FLAG"))
		t.Fatal("version flag did not exit")
	}

	for _, versionFlag := range []string{"-v", "--version"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestVersionFlagExitsBeforeDocker$")
		cmd.Env = append(os.Environ(),
			"DOCKTOP_VERSION_TEST=1",
			"DOCKTOP_VERSION_FLAG="+versionFlag,
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("%s exited with error: %v\n%s", versionFlag, err, output)
			continue
		}
		wantOutput := []string{"docktop "}
		if commit != "none" && commit != "" {
			wantOutput = append(wantOutput, "Commit:")
		}
		if date != "unknown" && date != "" {
			wantOutput = append(wantOutput, "Built at:")
		}
		for _, want := range wantOutput {
			if !strings.Contains(string(output), want) {
				t.Errorf("%s output %q does not contain %q", versionFlag, output, want)
			}
		}
	}
}

func parseFlagsForTest(t *testing.T, args ...string) cliOptions {
	t.Helper()

	originalArgs := os.Args
	originalCommandLine := flag.CommandLine
	originalUsage := flag.Usage
	t.Cleanup(func() {
		os.Args = originalArgs
		flag.CommandLine = originalCommandLine
		flag.Usage = originalUsage
	})

	flag.CommandLine = flag.NewFlagSet("docktop", flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = append([]string{"docktop"}, args...)
	return parseFlags()
}
