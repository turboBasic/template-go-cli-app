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
	if got.values["path"] != want {
		t.Errorf("resolved path = %q, want %q", got.values["path"], want)
	}
	if got.Sources["path"] != source {
		t.Errorf("path source = %q, want %q", got.Sources["path"], source)
	}
}

func assertPlainString(t *testing.T, got LoadResult) {
	t.Helper()
	if got.values["label"] != "plain/relative" {
		t.Errorf("ordinary string = %q, want unchanged plain/relative", got.values["label"])
	}
}
