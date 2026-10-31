package domain

import "errors"

var ErrResourceNotFound = errors.New("resource not found")

// Resource is a named infrastructure resource and its current status.
type Resource struct {
	Name   string
	Status string
}

// Validate checks the invariants required for a resource to be listed.
func (r Resource) Validate() error {
	if r.Name == "" {
		return errors.New("resource name is required")
	}
	if r.Status == "" {
		return errors.New("resource status is required")
	}
	return nil
}
