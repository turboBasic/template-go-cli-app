package app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

type fakeResourceReader struct {
	resources []domain.Resource
	err       error
}

func (r fakeResourceReader) List(context.Context) ([]domain.Resource, error) {
	return r.resources, r.err
}

func TestResourceServiceList(t *testing.T) {
	want := []domain.Resource{{Name: "example-1", Status: "ready"}}
	service := NewResourceService(fakeResourceReader{resources: want})

	got, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}
}

func TestResourceServiceListPropagatesReaderError(t *testing.T) {
	wantErr := errors.New("reader unavailable")
	service := NewResourceService(fakeResourceReader{err: wantErr})

	_, err := service.List(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("List() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestResourceServiceListRejectsInvalidResource(t *testing.T) {
	service := NewResourceService(fakeResourceReader{
		resources: []domain.Resource{{Status: "ready"}},
	})

	if _, err := service.List(context.Background()); err == nil {
		t.Fatal("List() error = nil, want invalid resource error")
	}
}
