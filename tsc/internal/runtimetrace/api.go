package runtimetrace

import (
	"context"
	"runtime/trace"
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

// Log emits a one-off event to the execution trace, attached to the task in
// ctx (if any). Category may be empty.
func Log(ctx context.Context, category, message string) {
	trace.Log(ctx, category, message)
}

// Logf is like Log but formats the message with fmt.Sprintf-style arguments.
func Logf(ctx context.Context, category, format string, args ...any) {
	trace.Logf(ctx, category, format, args...)
}
