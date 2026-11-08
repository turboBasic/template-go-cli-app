package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestExampleFileMatchesSchema keeps the root example file in step with the code: uncommenting every
// setting must give a valid file that lists each supported key, and the values it shows as defaults
// must be the defaults. The glob survives scripts/init.sh renaming the file.
func TestExampleFileMatchesSchema(t *testing.T) {
	paths, err := filepath.Glob("../../.*.example.toml")
	if err != nil || len(paths) != 1 {
		t.Fatalf("want exactly one root .*.example.toml, got %v (err %v)", paths, err)
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	var uncommented []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "# ") && line != "#" {
			line = line[1:]
		}
		uncommented = append(uncommented, line)
	}
	text := strings.Join(uncommented, "\n")

	values, err := parseTOML(paths[0], []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	listed := make(map[string]bool)
	for _, leaf := range leaves(values) {
		listed[leaf.key] = true
	}
	for key := range productionSchema() {
		if !listed[key] {
			t.Errorf("%s does not list key %q", paths[0], key)
		}
	}

	got, err := DecodeTOML(paths[0], []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	// These have no default; the file shows an example value for each.
	got.Log.File.Path = ""
	got.Timeouts.Demo = DemoTimeouts{}
	got.Demo.SecretCredential.Reference = ""
	if want := Defaults(); !reflect.DeepEqual(got, want) {
		t.Errorf("example file defaults differ from Defaults():\n got %+v\nwant %+v", got, want)
	}
}
