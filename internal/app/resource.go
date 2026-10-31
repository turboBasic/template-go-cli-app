package app

import (
	"context"
	"fmt"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

// ResourceReader supplies resources to application use cases.
type ResourceReader interface {
	List(context.Context) ([]domain.Resource, error)
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
