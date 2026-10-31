package memory

import (
	"context"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

// ResourceReader provides the reference resources from local memory.
type ResourceReader struct{}

var _ app.ResourceReader = ResourceReader{}

// List returns the fixed resources used by the reference application.
func (ResourceReader) List(ctx context.Context) ([]domain.Resource, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []domain.Resource{
		{Name: "example-1", Status: "ready"},
		{Name: "example-2", Status: "pending"},
	}, nil
}
