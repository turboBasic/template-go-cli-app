package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathOriginMatrix(t *testing.T) {
	t.Run("user file", func(t *testing.T) {
		file := configAt(t, t.TempDir(), "path = 'relative/file'\nlabel = 'plain/relative'\n")
		got := loadPathSchema(t, ConfigFiles{User: file}, nil, nil)
		assertPath(t, got, filepath.Join(filepath.Dir(file), "relative/file"), Source(file))
		assertPlainString(t, got)
	})

	t.Run("project file wins and supplies its own directory", func(t *testing.T) {
		root := t.TempDir()
		user := configAt(t, filepath.Join(root, "user"), "path = 'relative/file'\nlabel = 'plain/relative'\n")
		project := configAt(t, filepath.Join(root, "project"), "path = 'relative/file'\n")
		got := loadPathSchema(t, ConfigFiles{User: user, Project: project}, nil, nil)
		assertPath(t, got, filepath.Join(filepath.Dir(project), "relative/file"), Source(project))
		assertPlainString(t, got)
	})

	t.Run("explicit project file origin", func(t *testing.T) {
		file := configAt(t, filepath.Join(t.TempDir(), "explicit"), "path = 'relative/file'\n")
		got := loadPathSchema(t, ConfigFiles{Project: file}, nil, nil)
		assertPath(t, got, filepath.Join(filepath.Dir(file), "relative/file"), Source(file))
	})

	t.Run("environment uses working directory", func(t *testing.T) {
		work := t.TempDir()
		got := loadPathSchema(t, ConfigFiles{}, []string{"CLI_APP_PATH=relative/file"}, nil, work)
		assertPath(t, got, filepath.Join(work, "relative/file"), "CLI_APP_PATH")
	})

	t.Run("changed flag uses working directory", func(t *testing.T) {
		work := t.TempDir()
		flags := map[string]FlagValue{"path": {Value: "relative/file", Changed: true, Source: "--path"}}
		got := loadPathSchema(t, ConfigFiles{}, nil, flags, work)
		assertPath(t, got, filepath.Join(work, "relative/file"), "--path")
	})

	t.Run("absolute path is preserved", func(t *testing.T) {
		absolute := filepath.Join(t.TempDir(), "absolute/file")
		file := configAt(t, t.TempDir(), "path = '"+absolute+"'\n")
		got := loadPathSchema(t, ConfigFiles{Project: file}, nil, nil)
		assertPath(t, got, absolute, Source(file))
	})
}

func TestLoggingFilePathSourceResolution(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	user := configAt(t, filepath.Join(root, "user"), "[log.file]\npath = 'logs/user.jsonl'\n")
	project := configAt(t, filepath.Join(root, "project"), "[log.file]\npath = 'logs/project.jsonl'\n")
	schema := productionSchema()
	schema["log.file.path"] = pathValue

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
		if got.Config.Log.Console.Level != "warn" {
			t.Errorf("console level changed during path resolution: %q", got.Config.Log.Console.Level)
		}
	})

	t.Run("environment path uses working directory", func(t *testing.T) {
		got, err := loadWithSchema(ConfigFiles{Project: project}, []string{"CLI_APP_LOG_FILE_PATH=logs/env.jsonl"}, nil, schema, work)
		if err != nil {
			t.Fatal(err)
		}
		assertLoggingPath(t, got, filepath.Join(work, "logs/env.jsonl"), "CLI_APP_LOG_FILE_PATH")
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

func loadPathSchema(t *testing.T, files ConfigFiles, environment []string, flags map[string]FlagValue, workingDir ...string) LoadResult {
	t.Helper()
	work := t.TempDir()
	if len(workingDir) != 0 {
		work = workingDir[0]
	}
	schema := productionSchema()
	schema["path"] = pathValue
	schema["label"] = stringValue
	got, err := loadWithSchema(files, environment, flags, schema, work)
	if err != nil {
		t.Fatal(err)
	}
	return got
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

func assertPath(t *testing.T, got LoadResult, want string, source Source) {
	t.Helper()
	value, _ := nestedLookup(got.values, "path")
	if value != want {
		t.Errorf("resolved path = %q, want %q", value, want)
	}
	if got.Sources["path"] != source {
		t.Errorf("path source = %q, want %q", got.Sources["path"], source)
	}
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

func assertPlainString(t *testing.T, got LoadResult) {
	t.Helper()
	if got.values["label"] != "plain/relative" {
		t.Errorf("ordinary string = %q, want unchanged plain/relative", got.values["label"])
	}
}
