package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

func TestRendererText(t *testing.T) {
	resources := []domain.Resource{{Name: "example-1", Status: "ready"}, {Name: "example-2", Status: "pending"}}
	tests := []struct {
		name string
		run  func(*Renderer) error
		want string
	}{
		{
			name: "list",
			run:  func(renderer *Renderer) error { return renderer.RenderList(resources) },
			want: "NAME        STATUS\nexample-1   ready\nexample-2   pending\n",
		},
		{
			name: "resource",
			run:  func(renderer *Renderer) error { return renderer.RenderResource(resources[0]) },
			want: "example-1: ready\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			renderer := Renderer{Writer: &output, Format: FormatText}
			if err := test.run(&renderer); err != nil {
				t.Fatal(err)
			}
			if got := output.String(); got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRendererJSON(t *testing.T) {
	resources := []domain.Resource{{Name: "example-1", Status: "ready"}, {Name: "example-2", Status: "pending"}}
	tests := []struct {
		name  string
		run   func(*Renderer) error
		check func(*testing.T, []byte)
	}{
		{
			name: "list",
			run:  func(renderer *Renderer) error { return renderer.RenderList(resources) },
			check: func(t *testing.T, data []byte) {
				var got []map[string]string
				if err := json.Unmarshal(data, &got); err != nil {
					t.Fatal(err)
				}
				want := []map[string]string{{"name": "example-1", "status": "ready"}, {"name": "example-2", "status": "pending"}}
				if len(got) != len(want) {
					t.Fatalf("decoded list = %#v, want %#v", got, want)
				}
				for i := range want {
					if len(got[i]) != 2 || got[i]["name"] != want[i]["name"] || got[i]["status"] != want[i]["status"] {
						t.Fatalf("decoded item %d = %#v, want %#v", i, got[i], want[i])
					}
				}
			},
		},
		{
			name: "resource",
			run:  func(renderer *Renderer) error { return renderer.RenderResource(resources[0]) },
			check: func(t *testing.T, data []byte) {
				var got map[string]string
				if err := json.Unmarshal(data, &got); err != nil {
					t.Fatal(err)
				}
				if len(got) != 2 || got["name"] != "example-1" || got["status"] != "ready" {
					t.Fatalf("decoded resource = %#v", got)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var first, second bytes.Buffer
			if err := (Renderer{Writer: &first, Format: FormatJSON}).run(test.run); err != nil {
				t.Fatal(err)
			}
			if err := (Renderer{Writer: &second, Format: FormatJSON}).run(test.run); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(first.Bytes(), second.Bytes()) {
				t.Fatalf("repeated JSON differs: %q vs %q", first.String(), second.String())
			}
			test.check(t, first.Bytes())
		})
	}
}

func (r Renderer) run(render func(*Renderer) error) error { return render(&r) }
