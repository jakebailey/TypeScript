package tsgorun

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/execute"
	"github.com/microsoft/TypeScript/tsc/internal/osutil"
)

// RunMain runs the main tsc command line, dispatching to --lsp or --api
// subcommands as needed. It returns the process exit code.
// If defaultLibraryPath is empty, the bundled library path is used.
func RunMain(args []string, defaultLibraryPath string) int {
	core.ApplyDebugStackLimit()
	if len(args) > 0 {
		switch args[0] {
		case "--lsp":
			return RunLSP(args[1:])
		case "--api":
			return RunAPI(args[1:])
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	result := execute.CommandLine(ctx, NewSystem(defaultLibraryPath), args, nil)
	return int(result.Status)
}

// Main calls RunMain with the process arguments and exits.
func Main() {
	os.Exit(RunMain(osutil.Args()[1:], ""))
}
