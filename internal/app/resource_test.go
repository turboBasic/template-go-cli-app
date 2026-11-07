package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
	"github.com/turboBasic/template-go-cli-app/internal/logging"
)

type fakeResourceReader struct {
	resources []domain.Resource
	resource  domain.Resource
	err       error
	apply     func(context.Context, string) (domain.Resource, error)
	// credentials records the credential of every Apply call when set.
	credentials *[]logging.Secret
}

func (r fakeResourceReader) List(context.Context) ([]domain.Resource, error) {
	return r.resources, r.err
}

func (r fakeResourceReader) Status(context.Context, string) (domain.Resource, error) {
	return r.resource, r.err
}

func (r fakeResourceReader) Apply(ctx context.Context, name string, credential logging.Secret) (domain.Resource, error) {
	if r.credentials != nil {
		*r.credentials = append(*r.credentials, credential)
	}
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

type fakeResolver struct {
	calls []string
	value logging.Secret
	err   error
	block bool
}

func (r *fakeResolver) Resolve(ctx context.Context, reference string) (logging.Secret, error) {
	r.calls = append(r.calls, reference)
	if r.block {
		<-ctx.Done()
		return "", ctx.Err()
	}
	return r.value, r.err
}

func TestResourceServiceApplyCredential(t *testing.T) {
	const reference = "op://vault/item/field"
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, test := range []struct {
		name         string
		ctx          context.Context
		reference    string
		resolver     *fakeResolver
		noResolver   bool
		wantCalls    int
		wantApplied  []logging.Secret
		wantEvents   int
		wantErr      error
		notErr       error
		wantContains string
	}{
		{name: "unset", reference: "", resolver: &fakeResolver{value: "x"}, wantApplied: []logging.Secret{""}, wantEvents: 3},
		{name: "set", reference: reference, resolver: &fakeResolver{value: "resolved"}, wantCalls: 1, wantApplied: []logging.Secret{"resolved"}, wantEvents: 3},
		{name: "missing secret", reference: reference, resolver: &fakeResolver{err: Classify(ErrValidation, errors.New("item not found"))}, wantCalls: 1, wantErr: ErrValidation, notErr: ErrAuthentication},
		{name: "authentication failure", reference: reference, resolver: &fakeResolver{err: Classify(ErrAuthentication, errors.New("signed out"))}, wantCalls: 1, wantErr: ErrAuthentication, notErr: ErrValidation},
		{name: "no resolver", reference: reference, noResolver: true, wantErr: ErrValidation, wantContains: "no secret provider configured"},
		{name: "cancelled before the call", ctx: cancelled, reference: reference, resolver: &fakeResolver{value: "x"}, wantErr: context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := test.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			var applied []logging.Secret
			var events []ProgressEvent
			options := []ResourceOption{WithCredential(test.resolver, test.reference)}
			if test.noResolver {
				options = []ResourceOption{WithCredential(nil, test.reference)}
			}
			reader := fakeResourceReader{resource: domain.Resource{Name: "example-1", Status: "ready"}, credentials: &applied}
			service := NewResourceService(reader, slog.New(slog.NewTextHandler(io.Discard, nil)), progressRecorder{events: &events}, options...)

			_, err := service.Apply(ctx, "example-1")

			if test.resolver != nil && len(test.resolver.calls) != test.wantCalls {
				t.Fatalf("resolver calls = %v, want %d", test.resolver.calls, test.wantCalls)
			}
			if test.wantCalls > 0 && test.resolver.calls[0] != reference {
				t.Fatalf("resolver reference = %q, want %q", test.resolver.calls[0], reference)
			}
			if test.wantErr == nil && err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("Apply() error = %v, want %v", err, test.wantErr)
			}
			if test.notErr != nil && errors.Is(err, test.notErr) {
				t.Fatalf("Apply() error = %v, must not be %v", err, test.notErr)
			}
			if test.wantContains != "" && !strings.Contains(err.Error(), test.wantContains) {
				t.Fatalf("Apply() error = %q, want it to contain %q", err, test.wantContains)
			}
			if !reflect.DeepEqual(applied, test.wantApplied) || len(events) != test.wantEvents {
				t.Fatalf("reader credentials = %q, events = %d; want %q and %d events", applied, len(events), test.wantApplied, test.wantEvents)
			}
		})
	}
}

func TestResourceServiceApplyCredentialResolutionIsCancellable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resolver := &fakeResolver{block: true}
	service := NewResourceService(fakeResourceReader{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, WithCredential(resolver, "op://v/i/f"))
	time.AfterFunc(20*time.Millisecond, cancel)

	if _, err := service.Apply(ctx, "example-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Apply() error = %v, want context cancellation", err)
	}
}

func TestResourceServiceApplyCredentialNeverLogged(t *testing.T) {
	const value = "s3cr3t-value"
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	reader := fakeResourceReader{resource: domain.Resource{Name: "example-1", Status: "ready"}}
	service := NewResourceService(reader, logger, nil, WithCredential(&fakeResolver{value: value}, "op://v/i/f"))

	if _, err := service.Apply(context.Background(), "example-1"); err != nil {
		t.Fatal(err)
	}
	if logs.Len() == 0 || strings.Contains(logs.String(), value) {
		t.Fatalf("logs = %q, want output that never contains the resolved value", logs.String())
	}
}
