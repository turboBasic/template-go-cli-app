package cli

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files instead of comparing against them")

func TestMain(m *testing.M) {
	// Fang caches the terminal width on first use; pin it so help layout does not depend on the
	// terminal running the tests.
	if err := os.Setenv("__FANG_TEST_WIDTH", "80"); err != nil {
		panic(err)
	}
	flag.Parse()
	os.Exit(m.Run())
}

// assertGolden compares got with testdata/<name> byte for byte; -update rewrites the file.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, got, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (regenerate with -update)", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s differs from golden\n got: %q\nwant: %q\n(regenerate with -update if the change is intended)", path, got, want)
	}
}

// withoutTerminalStdout hides a real terminal from Fang, which asks it for the background color.
func withoutTerminalStdout(t *testing.T) {
	t.Helper()
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = null
	t.Cleanup(func() {
		os.Stdout = original
		_ = null.Close()
	})
}
