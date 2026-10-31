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
	resource  domain.Resource
	err       error
}

func (r fakeResourceReader) List(context.Context) ([]domain.Resource, error) {
	return r.resources, r.err
}

func (r fakeResourceReader) Status(context.Context, string) (domain.Resource, error) {
	return r.resource, r.err
}

func (r fakeResourceReader) Apply(context.Context, string) (domain.Resource, error) {
	return r.resource, r.err
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

func TestResourceServiceStatusAndApply(t *testing.T) {
	want := domain.Resource{Name: "example-2", Status: "ready"}
	service := NewResourceService(fakeResourceReader{resource: want})
	for name, useCase := range map[string]func(context.Context, string) (domain.Resource, error){
		"Status": service.Status,
		"Apply":  service.Apply,
	} {
		t.Run(name, func(t *testing.T) {
			got, err := useCase(context.Background(), want.Name)
			if err != nil || got != want {
				t.Fatalf("use case = (%#v, %v), want (%#v, nil)", got, err, want)
			}
		})
	}
}

func TestResourceServiceStatusAndApplyWrapAdapterError(t *testing.T) {
	wantErr := errors.New("adapter unavailable")
	service := NewResourceService(fakeResourceReader{err: wantErr})
	for name, useCase := range map[string]func(context.Context, string) (domain.Resource, error){
		"Status": service.Status,
		"Apply":  service.Apply,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := useCase(context.Background(), "example-1"); !errors.Is(err, wantErr) {
				t.Fatalf("error = %v, want wrapped %v", err, wantErr)
			}
		})
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
