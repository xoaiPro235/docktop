package docker

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDockerContextHost(t *testing.T) {
	t.Run("resolves custom active context", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("DOCKER_CONFIG", tmpDir)

		// 1. Write config.json
		configJSON := `{"currentContext": "desktop-linux"}`
		if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(configJSON), 0644); err != nil {
			t.Fatalf("failed to write config.json: %v", err)
		}

		// 2. Compute SHA256 of context name
		h := sha256.Sum256([]byte("desktop-linux"))
		contextHash := hex.EncodeToString(h[:])

		// 3. Write meta.json
		metaDir := filepath.Join(tmpDir, "contexts", "meta", contextHash)
		if err := os.MkdirAll(metaDir, 0755); err != nil {
			t.Fatalf("failed to create meta dir: %v", err)
		}

		expectedHost := "unix:///Users/test/.docker/run/docker.sock"
		metaJSON := `{
			"Name": "desktop-linux",
			"Endpoints": {
				"docker": {
					"Host": "` + expectedHost + `"
				}
			}
		}`
		if err := os.WriteFile(filepath.Join(metaDir, "meta.json"), []byte(metaJSON), 0644); err != nil {
			t.Fatalf("failed to write meta.json: %v", err)
		}

		// 4. Assert resolution
		got := resolveDockerContextHost()
		if got != expectedHost {
			t.Errorf("expected host %q, got %q", expectedHost, got)
		}

		// Also check resolveDefaultHost
		resolved := resolveDefaultHost()
		if resolved != expectedHost {
			t.Errorf("expected resolveDefaultHost %q, got %q", expectedHost, resolved)
		}
	})

	t.Run("resolves context via DOCKER_CONTEXT env override", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("DOCKER_CONFIG", tmpDir)
		t.Setenv("DOCKER_CONTEXT", "orbstack")

		// Even if config.json has something else, DOCKER_CONTEXT should take precedence
		configJSON := `{"currentContext": "desktop-linux"}`
		if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(configJSON), 0644); err != nil {
			t.Fatalf("failed to write config.json: %v", err)
		}

		h := sha256.Sum256([]byte("orbstack"))
		contextHash := hex.EncodeToString(h[:])

		metaDir := filepath.Join(tmpDir, "contexts", "meta", contextHash)
		if err := os.MkdirAll(metaDir, 0755); err != nil {
			t.Fatalf("failed to create meta dir: %v", err)
		}

		expectedHost := "unix:///Users/test/.orbstack/run/docker.sock"
		metaJSON := `{
			"Name": "orbstack",
			"Endpoints": {
				"docker": {
					"Host": "` + expectedHost + `"
				}
			}
		}`
		if err := os.WriteFile(filepath.Join(metaDir, "meta.json"), []byte(metaJSON), 0644); err != nil {
			t.Fatalf("failed to write meta.json: %v", err)
		}

		got := resolveDockerContextHost()
		if got != expectedHost {
			t.Errorf("expected host %q from DOCKER_CONTEXT, got %q", expectedHost, got)
		}
	})

	t.Run("returns empty when default context", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("DOCKER_CONFIG", tmpDir)

		configJSON := `{"currentContext": "default"}`
		if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(configJSON), 0644); err != nil {
			t.Fatalf("failed to write config.json: %v", err)
		}

		got := resolveDockerContextHost()
		if got != "" {
			t.Errorf("expected empty string for default context, got %q", got)
		}
	})

	t.Run("returns empty when config does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("DOCKER_CONFIG", tmpDir)

		got := resolveDockerContextHost()
		if got != "" {
			t.Errorf("expected empty string when no config.json, got %q", got)
		}
	})

	t.Run("returns empty when meta.json is missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("DOCKER_CONFIG", tmpDir)

		configJSON := `{"currentContext": "missing-context"}`
		if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(configJSON), 0644); err != nil {
			t.Fatalf("failed to write config.json: %v", err)
		}

		got := resolveDockerContextHost()
		if got != "" {
			t.Errorf("expected empty string when meta.json is missing, got %q", got)
		}
	})
}
