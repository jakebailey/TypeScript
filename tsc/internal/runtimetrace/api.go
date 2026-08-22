package runtimetrace

import (
	"context"
	"runtime/trace"
	"sync/atomic"
)

// IsEnabled reports whether the runtime execution tracer is enabled. It is
// thin wrapper over runtime/trace.IsEnabled and is provided so callers can
// avoid an extra import.
func IsEnabled() bool { return trace.IsEnabled() }

// Region starts a region in the calling goroutine and returns a function that
// ends it. Designed to be used with defer:
//
//	defer runtimetrace.Region(ctx, "parse")()
//
// If a task is attached to ctx (via NewTask) the region is associated with
// it; otherwise the region is attached to the background task. When the
// runtime tracer is disabled, both StartRegion and Region.End are cheap
// no-ops.
func Region(ctx context.Context, name string) func() {
	return trace.StartRegion(ctx, name).End
}

// NewTask creates a new task and returns a derived context that carries it
// along with a function that ends the task. Designed to be used with defer:
//
//	ctx, end := runtimetrace.NewTask(ctx, "lsp.textDocument/completion")
//	defer end()
//
// Tasks are higher-level than regions: they group regions and log events
// across goroutines and produce a latency entry in the trace's task table.
func NewTask(ctx context.Context, name string) (context.Context, func()) {
	ctx, task := trace.NewTask(ctx, name)
	return ctx, task.End
}

// --- Logging --------------------------------------------------------------
//
// Two logging variants are provided to make it explicit at the call site
// whether the payload is safe to share:
//
//   - LogSafe / LogSafef: for payloads known to contain no user data.
//     Examples: counts, sizes, durations, protocol method names, JSON-RPC
//     request IDs, enum values. Always emitted when tracing is on.
//
//   - LogUnsafe / LogUnsafef: for payloads that may contain user data such
//     as file paths, module specifiers, or identifier names. Only emitted
//     when the user has opted in via TSGO_RUNTIME_TRACE_DETAIL.
//
// Use UnsafeLoggingEnabled to short-circuit expensive payload construction
// before calling the LogUnsafe* helpers.

// unsafeLogging is set by Start when TSGO_RUNTIME_TRACE_DETAIL is truthy.
var unsafeLogging atomic.Bool

// UnsafeLoggingEnabled reports whether unsafe (potentially user-data-bearing)
// trace logging has been enabled by the user.
func UnsafeLoggingEnabled() bool { return unsafeLogging.Load() }

// LogSafe emits a one-off event to the execution trace, attached to the task
// in ctx (if any). The payload must not contain user data.
func LogSafe(ctx context.Context, category, message string) {
	trace.Log(ctx, category, message)
}

// LogSafef is like LogSafe but formats the message with fmt.Sprintf-style
// arguments. The formatted payload must not contain user data.
func LogSafef(ctx context.Context, category, format string, args ...any) {
	trace.Logf(ctx, category, format, args...)
}

// LogUnsafe emits a one-off event to the execution trace only when the user
// has opted in via TSGO_RUNTIME_TRACE_DETAIL. Use it for payloads that may
// include file paths, identifier names, or other user data.
func LogUnsafe(ctx context.Context, category, message string) {
	if unsafeLogging.Load() {
		trace.Log(ctx, category, message)
	}
}

// LogUnsafef is like LogUnsafe but formats the message with fmt.Sprintf-style
// arguments.
func LogUnsafef(ctx context.Context, category, format string, args ...any) {
	if unsafeLogging.Load() {
		trace.Logf(ctx, category, format, args...)
	}
}
