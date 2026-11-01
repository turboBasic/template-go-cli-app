package app

import (
	"errors"
	"fmt"
	"testing"
)

type testCause struct{ message string }

func (e *testCause) Error() string { return e.message }

func TestClassifyPreservesCategoryAndCause(t *testing.T) {
	categories := []error{
		ErrValidation,
		ErrAuthentication,
		ErrAuthorization,
		ErrConflict,
		ErrUnavailable,
		ErrTimeout,
		ErrCancellation,
	}
	for _, category := range categories {
		t.Run(category.Error(), func(t *testing.T) {
			wantCause := &testCause{message: "underlying failure"}
			err := fmt.Errorf("operation failed: %w", Classify(category, wantCause))
			if !errors.Is(err, category) {
				t.Fatalf("error %v does not match category %v", err, category)
			}
			var gotCause *testCause
			if !errors.As(err, &gotCause) || gotCause != wantCause {
				t.Fatalf("errors.As() cause = %#v, want %#v", gotCause, wantCause)
			}
		})
	}
}

func TestClassifyNil(t *testing.T) {
	if err := Classify(ErrValidation, nil); err != nil {
		t.Fatalf("Classify(nil) = %v, want nil", err)
	}
}
