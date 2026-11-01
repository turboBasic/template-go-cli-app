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
	if err := ctx.Err(); err != nil {
		return domain.Resource{}, fmt.Errorf("apply resource: %w", classifyDeadline(err))
	}
	s.reportProgress(ProgressEvent{Phase: "validating", Message: "Validating resource"})
	if err := ctx.Err(); err != nil {
		return domain.Resource{}, fmt.Errorf("apply resource: %w", classifyDeadline(err))
	}
	s.reportProgress(ProgressEvent{Phase: "applying", Message: "Applying resource"})
	if err := ctx.Err(); err != nil {
		return domain.Resource{}, fmt.Errorf("apply resource: %w", classifyDeadline(err))
	}
	resource, err := s.reader.Apply(ctx, name)
	if err != nil {
		return domain.Resource{}, fmt.Errorf("apply resource: %w", classifyDeadline(err))
	}
	if err := ctx.Err(); err != nil {
		return domain.Resource{}, fmt.Errorf("apply resource: %w", classifyDeadline(err))
	}
	if err := resource.Validate(); err != nil {
		return domain.Resource{}, fmt.Errorf("validate applied resource: %w", Classify(ErrValidation, err))
	}
	if err := ctx.Err(); err != nil {
		return domain.Resource{}, fmt.Errorf("apply resource: %w", classifyDeadline(err))
	}
	s.reportProgress(ProgressEvent{Phase: "complete", Current: 1, Total: 1, Message: "Resource applied"})
	return resource, nil
}

func (s *ResourceService) reportProgress(event ProgressEvent) {
	if s.progressReporter != nil {
		s.progressReporter.Report(event)
	}
}

// ResourceService coordinates resource use cases.
type ResourceService struct {
	reader           ResourceReader
	logger           *slog.Logger
	progressReporter ProgressReporter
}

// NewResourceService constructs a resource service with its required dependencies.
func NewResourceService(reader ResourceReader, logger *slog.Logger, progressReporter ProgressReporter) *ResourceService {
	return &ResourceService{reader: reader, logger: logger, progressReporter: progressReporter}
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
