package app

import (
	"context"
	"fmt"
	"log/slog"

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
	s.logger.DebugContext(ctx, "getting resource status")
	resource, err := s.reader.Status(ctx, name)
	if err != nil {
		return domain.Resource{}, fmt.Errorf("get resource status: %w", classifyDeadline(err))
	}
	if err := resource.Validate(); err != nil {
		return domain.Resource{}, fmt.Errorf("validate resource status: %w", Classify(ErrValidation, err))
	}
	return resource, nil
}

// Apply applies the named resource and returns its updated state.
func (s *ResourceService) Apply(ctx context.Context, name string) (domain.Resource, error) {
	s.logger.DebugContext(ctx, "applying resource")
	resource, err := s.reader.Apply(ctx, name)
	if err != nil {
		return domain.Resource{}, fmt.Errorf("apply resource: %w", classifyDeadline(err))
	}
	if err := resource.Validate(); err != nil {
		return domain.Resource{}, fmt.Errorf("validate applied resource: %w", Classify(ErrValidation, err))
	}
	return resource, nil
}

// ResourceService coordinates resource use cases.
type ResourceService struct {
	reader ResourceReader
	logger *slog.Logger
}

// NewResourceService constructs a resource service with its required reader and logger.
func NewResourceService(reader ResourceReader, logger *slog.Logger) *ResourceService {
	return &ResourceService{reader: reader, logger: logger}
}

// List returns the resources supplied by the configured reader.
func (s *ResourceService) List(ctx context.Context) ([]domain.Resource, error) {
	s.logger.DebugContext(ctx, "listing resources")
	resources, err := s.reader.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", classifyDeadline(err))
	}
	for i, resource := range resources {
		if err := resource.Validate(); err != nil {
			return nil, fmt.Errorf("resource %d: %w", i, Classify(ErrValidation, err))
		}
	}
	return resources, nil
}
