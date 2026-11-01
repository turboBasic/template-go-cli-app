package app

// ProgressEvent describes a synchronous application progress update.
// A zero Total means progress is indeterminate. A positive Total with
// Current greater than or equal to Total is measurable completion. The
// "complete" phase is reserved for terminal completion, including work with
// an indeterminate total; Message carries the user-facing completion text.
type ProgressEvent struct {
	Phase   string
	Current int
	Total   int
	Message string
}

// ProgressReporter synchronously reports application progress; Report returns
// only after the reporting attempt completes. Implementations should keep
// reporting best-effort and must not require rendering or CLI dependencies.
type ProgressReporter interface {
	Report(ProgressEvent)
}
