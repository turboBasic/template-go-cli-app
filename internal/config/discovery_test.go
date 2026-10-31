package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDiscovery(t *testing.T) {
	t.Run("XDG user and project files", func(t *testing.T) {
		root := t.TempDir()
		work := filepath.Join(root, "work")
		xdg := filepath.Join(root, "xdg")
		writeConfig(t, filepath.Join(work, ".cli-app.toml"))
		writeConfig(t, filepath.Join(xdg, "cli-app", "config.toml"))
		t.Setenv("XDG_CONFIG_HOME", xdg)

		got, err := DiscoverFiles(work, "cli-app", "")
		if err != nil {
			t.Fatal(err)
		}
		want := ConfigFiles{User: filepath.Join(xdg, "cli-app", "config.toml"), Project: filepath.Join(work, ".cli-app.toml")}
		if got != want {
			t.Fatalf("DiscoverFiles() = %+v, want %+v", got, want)
		}
	})

	t.Run("home fallback", func(t *testing.T) {
		home := t.TempDir()
		work := t.TempDir()
		user := filepath.Join(home, ".config", "cli-app", "config.toml")
		writeConfig(t, user)
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", home)

		got, err := DiscoverFiles(work, "cli-app", "")
		if err != nil {
			t.Fatal(err)
		}
		if got.User != user {
			t.Fatalf("user config = %q, want %q", got.User, user)
		}
	})

	t.Run("absent optional files", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "missing"))
		got, err := DiscoverFiles(t.TempDir(), "cli-app", "")
		if err != nil {
			t.Fatal(err)
		}
		if got != (ConfigFiles{}) {
			t.Fatalf("DiscoverFiles() = %+v, want no files", got)
		}
	})

	t.Run("relative explicit file selects project", func(t *testing.T) {
		work := t.TempDir()
		xdg := filepath.Join(t.TempDir(), "xdg")
		explicit := filepath.Join(work, "configs", "local.toml")
		user := filepath.Join(xdg, "cli-app", "config.toml")
		writeConfig(t, explicit)
		writeConfig(t, user)
		writeConfig(t, filepath.Join(work, ".cli-app.toml"))
		t.Setenv("XDG_CONFIG_HOME", xdg)

		got, err := DiscoverFiles(work, "cli-app", filepath.Join("configs", "local.toml"))
		if err != nil {
			t.Fatal(err)
		}
		if got != (ConfigFiles{User: user, Project: explicit}) {
			t.Fatalf("DiscoverFiles() = %+v, want user %q and explicit project %q", got, user, explicit)
		}
	})

	t.Run("missing explicit file fails", func(t *testing.T) {
		work := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "missing"))
		_, err := DiscoverFiles(work, "cli-app", "missing.toml")
		if err == nil || !strings.Contains(err.Error(), "missing.toml") || !strings.Contains(err.Error(), "does not exist") {
			t.Fatalf("DiscoverFiles() error = %v, want actionable missing-file error", err)
		}
	})

	t.Run("parent project file is not discovered", func(t *testing.T) {
		parent := t.TempDir()
		work := filepath.Join(parent, "child")
		if err := os.Mkdir(work, 0o755); err != nil {
			t.Fatal(err)
		}
		writeConfig(t, filepath.Join(parent, ".cli-app.toml"))
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "missing"))

		got, err := DiscoverFiles(work, "cli-app", "")
		if err != nil {
			t.Fatal(err)
		}
		if got.Project != "" {
			t.Fatalf("project config = %q, want none", got.Project)
		}
	})
}

func writeConfig(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
}
