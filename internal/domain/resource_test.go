package domain

import "testing"

func TestResourceValidate(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
		wantErr  bool
	}{
		{name: "valid", resource: Resource{Name: "example-1", Status: "ready"}},
		{name: "missing name", resource: Resource{Status: "ready"}, wantErr: true},
		{name: "missing status", resource: Resource{Name: "example-1"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.resource.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
