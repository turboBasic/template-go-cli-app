package output

import (
	"strings"
	"testing"
)

func TestStylesStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     string
		wantPrefix string
	}{
		{name: "ready is green", status: "ready", wantPrefix: "\x1b[32m"},
		{name: "pending is yellow", status: "pending", wantPrefix: "\x1b[33m"},
		{name: "unknown stays unstyled", status: "failed", wantPrefix: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := newStyles().status(tt.status)
			if tt.wantPrefix == "" {
				if got != tt.status {
					t.Fatalf("status(%q) = %q, want unchanged", tt.status, got)
				}
				return
			}
			if !strings.HasPrefix(got, tt.wantPrefix) || !strings.Contains(got, tt.status) {
				t.Fatalf("status(%q) = %q, want prefix %q around the text", tt.status, got, tt.wantPrefix)
			}
		})
	}
}

func TestStylesHeaderIsBold(t *testing.T) {
	t.Parallel()

	if got := newStyles().header.Render("NAME"); !strings.HasPrefix(got, "\x1b[1m") {
		t.Fatalf("header = %q, want bold prefix", got)
	}
}
