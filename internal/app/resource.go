package app

import (
	"context"
	"fmt"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

// ResourceReader supplies resources to application use cases.
type ResourceReader interface {
	List(context.Context) ([]domain.Resource, error)
	Status(context.Context, string) (domain.Resource, error)
	Apply(context.Context, string) (domain.Resource, error)
}

// Status returns the named resource and its current status.
func (s *ResourceService) Status(ctx context.Context, name string) (domain.Resource, error) {
	resource, err := s.reader.Status(ctx, name)
	if err != nil {
		return domain.Resource{}, fmt.Errorf("get resource status: %w", err)
	}
	if err := resource.Validate(); err != nil {
		return domain.Resource{}, fmt.Errorf("validate resource status: %w", err)
	}
	return resource, nil
}

// Apply applies the named resource and returns its updated state.
func (s *ResourceService) Apply(ctx context.Context, name string) (domain.Resource, error) {
	resource, err := s.reader.Apply(ctx, name)
	if err != nil {
		return domain.Resource{}, fmt.Errorf("apply resource: %w", err)
	}
	if err := resource.Validate(); err != nil {
		return domain.Resource{}, fmt.Errorf("validate applied resource: %w", err)
	}
	return resource, nil
}

// ResourceService coordinates resource use cases.
type ResourceService struct {
	reader ResourceReader
}

// NewResourceService constructs a resource service with its required reader.
func NewResourceService(reader ResourceReader) *ResourceService {
	return &ResourceService{reader: reader}
}

// List returns the resources supplied by the configured reader.
func (s *ResourceService) List(ctx context.Context) ([]domain.Resource, error) {
	resources, err := s.reader.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	for i, resource := range resources {
		if err := resource.Validate(); err != nil {
			return nil, fmt.Errorf("resource %d: %w", i, err)
		}
	}
	return resources, nil
}
