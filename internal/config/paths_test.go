package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoggingFilePathSourceResolution(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	user := configAt(t, filepath.Join(root, "user"), "[log.file]\npath = 'logs/user.jsonl'\n")
	project := configAt(t, filepath.Join(root, "project"), "[log.file]\npath = 'logs/project.jsonl'\n")
	schema := productionSchema()

	t.Run("user TOML origin", func(t *testing.T) {
		got, err := loadWithSchema(ConfigFiles{User: user}, nil, nil, schema, work)
		if err != nil {
			t.Fatal(err)
		}
		assertLoggingPath(t, got, filepath.Join(filepath.Dir(user), "logs/user.jsonl"), Source(user))
	})

	t.Run("project TOML origin wins", func(t *testing.T) {
		got, err := loadWithSchema(ConfigFiles{User: user, Project: project}, nil, nil, schema, work)
		if err != nil {
			t.Fatal(err)
		}
		assertLoggingPath(t, got, filepath.Join(filepath.Dir(project), "logs/project.jsonl"), Source(project))
		if got.Config.Log.Level != "warn" {
			t.Errorf("console level changed during path resolution: %q", got.Config.Log.Level)
		}
	})

	t.Run("environment path uses working directory", func(t *testing.T) {
		got, err := loadWithSchema(ConfigFiles{Project: project}, []string{"CLI_APP_LOG_FILE_PATH=logs/env.jsonl"}, nil, schema, work)
		if err != nil {
			t.Fatal(err)
		}
		assertLoggingPath(t, got, filepath.Join(work, "logs/env.jsonl"), "CLI_APP_LOG_FILE_PATH")
	})

	t.Run("changed flag uses working directory", func(t *testing.T) {
		flags := map[string]FlagValue{"log-file-path": {Value: "logs/flag.jsonl", Changed: true, Source: "--log-file-path"}}
		got, err := loadWithSchema(ConfigFiles{Project: project}, nil, flags, schema, work)
		if err != nil {
			t.Fatal(err)
		}
		assertLoggingPath(t, got, filepath.Join(work, "logs/flag.jsonl"), "--log-file-path")
	})

	t.Run("absolute path is preserved", func(t *testing.T) {
		absolute := filepath.Join(root, "absolute.jsonl")
		file := configAt(t, filepath.Join(root, "absolute"), "[log.file]\npath = '"+absolute+"'\n")
		got, err := loadWithSchema(ConfigFiles{Project: file}, nil, nil, schema, work)
		if err != nil {
			t.Fatal(err)
		}
		assertLoggingPath(t, got, absolute, Source(file))
	})

	t.Run("unset file path remains unset", func(t *testing.T) {
		got, err := loadWithSchema(ConfigFiles{}, nil, nil, schema, work)
		if err != nil {
			t.Fatal(err)
		}
		if got.Config.Log.File.Path != "" {
			t.Fatalf("default log file path = %q, want empty", got.Config.Log.File.Path)
		}
	})
}

func configAt(t *testing.T, dir, contents string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.toml")
	writeFile(t, path, contents)
	return path
}

func assertLoggingPath(t *testing.T, got LoadResult, want string, source Source) {
	t.Helper()
	if got.Config.Log.File.Path != want {
		t.Errorf("logging file path = %q, want %q", got.Config.Log.File.Path, want)
	}
	if got.Sources["log.file.path"] != source {
		t.Errorf("logging file path source = %q, want %q", got.Sources["log.file.path"], source)
	}
}
