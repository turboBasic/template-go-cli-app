package retry

import (
	"context"
	"errors"
	"fmt"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

var _ app.ResourceReader = (*ResourceReader)(nil)

// ResourceReader retries only explicitly configured, read-only status lookups.
type ResourceReader struct {
	reader app.ResourceReader
	status *Policy
	clock  Clock
	random Random
}

// NewResourceReader validates and wraps a resource reader with optional status retries.
func NewResourceReader(reader app.ResourceReader, status *Policy, clock Clock, random Random) (*ResourceReader, error) {
	if reader == nil {
		return nil, errors.New("resource reader is required")
	}
	if status != nil {
		if err := status.Validate(); err != nil {
			return nil, fmt.Errorf("validate status retry policy: %w", err)
		}
		if status.MaxAttempts > 1 && (clock == nil || status.Jitter > 0 && random == nil) {
			return nil, errors.New("status retries require a clock and jitter requires a random source")
		}
	}
	return &ResourceReader{reader: reader, status: status, clock: clock, random: random}, nil
}

// List delegates without retrying the operation.
func (r *ResourceReader) List(ctx context.Context) ([]domain.Resource, error) {
	return r.reader.List(ctx)
}

// Status retries only unavailable errors because this operation is read-only.
func (r *ResourceReader) Status(ctx context.Context, name string) (domain.Resource, error) {
	var resource domain.Resource
	err := Do(ctx, func(ctx context.Context) error {
		var err error
		resource, err = r.reader.Status(ctx, name)
		return err
	}, r.status, Idempotent, func(err error) bool { return errors.Is(err, app.ErrUnavailable) }, r.clock, r.random)
	return resource, err
}

// Apply delegates without retrying the mutating operation.
func (r *ResourceReader) Apply(ctx context.Context, name string) (domain.Resource, error) {
	return r.reader.Apply(ctx, name)
}
