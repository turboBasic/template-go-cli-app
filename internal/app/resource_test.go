package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
)

type fakeResourceReader struct {
	resources []domain.Resource
	resource  domain.Resource
	err       error
	apply     func(context.Context, string) (domain.Resource, error)
}

func (r fakeResourceReader) List(context.Context) ([]domain.Resource, error) {
	return r.resources, r.err
}

func (r fakeResourceReader) Status(context.Context, string) (domain.Resource, error) {
	return r.resource, r.err
}

func (r fakeResourceReader) Apply(ctx context.Context, name string) (domain.Resource, error) {
	if r.apply != nil {
		return r.apply(ctx, name)
	}
	return r.resource, r.err
}

func TestResourceServiceList(t *testing.T) {
	want := []domain.Resource{{Name: "example-1", Status: "ready"}}
	service := newTestResourceService(fakeResourceReader{resources: want})

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
	service := newTestResourceService(fakeResourceReader{resource: want})
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

func TestResourceServiceApplyReportsProgress(t *testing.T) {
	wantResource := domain.Resource{Name: "example-1", Status: "ready"}
	var gotEvents []ProgressEvent
	service := newTestResourceService(fakeResourceReader{resource: wantResource}, progressRecorder{events: &gotEvents})
	got, err := service.Apply(context.Background(), wantResource.Name)
	if err != nil || got != wantResource {
		t.Fatalf("Apply() = (%#v, %v), want (%#v, nil)", got, err, wantResource)
	}
	wantEvents := []ProgressEvent{
		{Phase: "applying", Message: "Applying resource"},
		{Phase: "validating", Message: "Validating resource"},
		{Phase: "complete", Current: 1, Total: 1, Message: "Resource applied"},
	}
	if !reflect.DeepEqual(gotEvents, wantEvents) {
		t.Fatalf("progress events = %#v, want %#v", gotEvents, wantEvents)
	}
}

func TestResourceServiceApplyDoesNotReportCompletionAfterFailure(t *testing.T) {
	wantErr := errors.New("adapter failed")
	var gotEvents []ProgressEvent
	service := newTestResourceService(fakeResourceReader{err: wantErr}, progressRecorder{events: &gotEvents})
	if _, err := service.Apply(context.Background(), "example-1"); !errors.Is(err, wantErr) {
		t.Fatalf("Apply() error = %v, want wrapped %v", err, wantErr)
	}
	if got, want := gotEvents, []ProgressEvent{
		{Phase: "applying", Message: "Applying resource"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("progress events = %#v, want %#v", got, want)
	}
}

func TestResourceServiceApplyChecksCancellationBetweenReportedPhases(t *testing.T) {
	for _, cancelPhase := range []string{"applying", "validating"} {
		t.Run(cancelPhase, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var gotEvents []ProgressEvent
			applyCalls := 0
			reader := fakeResourceReader{resource: domain.Resource{Name: "example-1", Status: "ready"}, apply: func(ctx context.Context, name string) (domain.Resource, error) {
				applyCalls++
				return domain.Resource{Name: name, Status: "ready"}, nil
			}}
			reporter := progressRecorder{events: &gotEvents, onReport: func(event ProgressEvent) {
				if event.Phase == cancelPhase {
					cancel()
				}
			}}
			service := newTestResourceService(reader, reporter)
			if _, err := service.Apply(ctx, "example-1"); !errors.Is(err, context.Canceled) {
				t.Fatalf("Apply() error = %v, want context cancellation", err)
			}
			wantEvents := []ProgressEvent{{Phase: "applying", Message: "Applying resource"}}
			if cancelPhase == "validating" {
				wantEvents = append(wantEvents, ProgressEvent{Phase: "validating", Message: "Validating resource"})
			}
			wantApplyCalls := 0
			if cancelPhase == "validating" {
				wantApplyCalls = 1
			}
			if !reflect.DeepEqual(gotEvents, wantEvents) || applyCalls != wantApplyCalls {
				t.Fatalf("progress events = %#v, apply calls = %d; want %#v and %d calls", gotEvents, applyCalls, wantEvents, wantApplyCalls)
			}
		})
	}
}

func TestResourceServiceApplyDoesNotReportCompletionAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var gotEvents []ProgressEvent
	reader := fakeResourceReader{apply: func(ctx context.Context, name string) (domain.Resource, error) {
		cancel()
		return domain.Resource{Name: name, Status: "ready"}, nil
	}}
	service := newTestResourceService(reader, progressRecorder{events: &gotEvents})
	if _, err := service.Apply(ctx, "example-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Apply() error = %v, want context cancellation", err)
	}
	if len(gotEvents) != 1 || gotEvents[0].Phase != "applying" {
		t.Fatalf("progress events = %#v, want no completion after cancellation", gotEvents)
	}
}

func TestResourceServiceStatusAndApplyWrapAdapterError(t *testing.T) {
	wantErr := errors.New("adapter unavailable")
	service := newTestResourceService(fakeResourceReader{err: wantErr})
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
	service := newTestResourceService(fakeResourceReader{err: wantErr})

	_, err := service.List(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("List() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestResourceServiceListRejectsInvalidResource(t *testing.T) {
	service := newTestResourceService(fakeResourceReader{
		resources: []domain.Resource{{Status: "ready"}},
	})

	if _, err := service.List(context.Background()); err == nil {
		t.Fatal("List() error = nil, want invalid resource error")
	}
}

type progressRecorder struct {
	events   *[]ProgressEvent
	onReport func(ProgressEvent)
}

func (r progressRecorder) Report(event ProgressEvent) {
	*r.events = append(*r.events, event)
	if r.onReport != nil {
		r.onReport(event)
	}
}

func newTestResourceService(reader ResourceReader, progressReporters ...ProgressReporter) *ResourceService {
	var progressReporter ProgressReporter
	if len(progressReporters) > 0 {
		progressReporter = progressReporters[0]
	}
	return NewResourceService(reader, slog.New(slog.NewTextHandler(io.Discard, nil)), progressReporter)
}
