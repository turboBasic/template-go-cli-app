package memory

import (
	"context"
	"time"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

// ResourceReader provides the reference resources from local memory.
type ResourceReader struct {
	resources  []domain.Resource
	applyDelay time.Duration
}

// Option configures a ResourceReader.
type Option func(*ResourceReader)

// WithApplyDelay makes Apply wait d, or return the context's error first, so a demo can show
// progress while an operation runs.
func WithApplyDelay(d time.Duration) Option {
	return func(r *ResourceReader) { r.applyDelay = d }
}

var _ app.ResourceReader = (*ResourceReader)(nil)

// NewResourceReader constructs the local reference fixtures.
func NewResourceReader(options ...Option) *ResourceReader {
	r := &ResourceReader{resources: []domain.Resource{
		{Name: "example-1", Status: "ready"},
		{Name: "example-2", Status: "pending"},
	}}
	for _, option := range options {
		option(r)
	}
	return r
}

// Names returns the names available for local resource completion.
func (r *ResourceReader) Names() []string {
	names := make([]string, len(r.resources))
	for i, resource := range r.resources {
		names[i] = resource.Name
	}
	return names
}

// List returns the fixed resources used by the reference application.
func (r *ResourceReader) List(ctx context.Context) ([]domain.Resource, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]domain.Resource(nil), r.resources...), nil
}

// Status returns a local fixture by name.
func (r *ResourceReader) Status(ctx context.Context, name string) (domain.Resource, error) {
	if err := ctx.Err(); err != nil {
		return domain.Resource{}, err
	}
	for _, resource := range r.resources {
		if resource.Name == name {
			return resource, nil
		}
	}
	return domain.Resource{}, domain.ErrResourceNotFound
}

// Apply marks a local fixture ready.
func (r *ResourceReader) Apply(ctx context.Context, name string) (domain.Resource, error) {
	if err := ctx.Err(); err != nil {
		return domain.Resource{}, err
	}
	if r.applyDelay > 0 {
		timer := time.NewTimer(r.applyDelay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return domain.Resource{}, ctx.Err()
		}
	}
	for i := range r.resources {
		if r.resources[i].Name == name {
			r.resources[i].Status = "ready"
			return r.resources[i], nil
		}
	}
	return domain.Resource{}, domain.ErrResourceNotFound
}
