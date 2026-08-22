//go:build windows

package ipc

import (
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"
)

// newPipeListener creates a Windows named pipe listener.
func newPipeListener(path string) (net.Listener, error) {
	return winio.ListenPipe(path, nil)
}

// GeneratePipePath returns a platform-appropriate pipe path for the given name.
func GeneratePipePath(name string) string {
	return `\\.\pipe\` + name
}

// NewFIFOTransport returns an error on Windows; FIFOs are not supported.
func NewFIFOTransport(_ string) (Transport, error) {
	return nil, fmt.Errorf("FIFO transport is not supported on Windows")
}
