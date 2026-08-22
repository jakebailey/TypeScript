//go:build !windows

package ipc

import (
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"sync"
	"syscall"
)

// newPipeListener creates a Unix domain socket listener.
func newPipeListener(path string) (net.Listener, error) {
	// Remove any existing socket file
	_ = os.Remove(path) //nolint:forbidigo
	return net.Listen("unix", path)
}

// GeneratePipePath returns a platform-appropriate pipe path for the given name.
func GeneratePipePath(name string) string {
	//nolint:forbidigo
	return path.Join(os.TempDir(), name)
}

// NewFIFOTransport creates two POSIX FIFOs at prefix.in and prefix.out.
func NewFIFOTransport(prefix string) (Transport, error) {
	inPath := prefix + ".in"
	outPath := prefix + ".out"

	if err := syscall.Mkfifo(inPath, 0o600); err != nil {
		return nil, fmt.Errorf("failed to create FIFO %s: %w", inPath, err)
	}
	if err := syscall.Mkfifo(outPath, 0o600); err != nil {
		_ = os.Remove(inPath) //nolint:forbidigo
		return nil, fmt.Errorf("failed to create FIFO %s: %w", outPath, err)
	}

	return &fifoTransport{prefix: prefix}, nil
}

type fifoTransport struct {
	prefix string
	once   sync.Once
	used   bool
}

func (t *fifoTransport) Accept() (io.ReadWriteCloser, error) {
	if t.used {
		return nil, io.EOF
	}
	t.used = true

	outFile, err := os.OpenFile(t.prefix+".out", os.O_WRONLY, 0) //nolint:forbidigo
	if err != nil {
		return nil, fmt.Errorf("failed to open FIFO %s.out for writing: %w", t.prefix, err)
	}

	inFile, err := os.OpenFile(t.prefix+".in", os.O_RDONLY, 0) //nolint:forbidigo
	if err != nil {
		outFile.Close() //nolint:forbidigo
		return nil, fmt.Errorf("failed to open FIFO %s.in for reading: %w", t.prefix, err)
	}

	return &fifoConn{reader: inFile, writer: outFile}, nil
}

func (t *fifoTransport) Close() error {
	t.once.Do(func() {
		_ = os.Remove(t.prefix + ".in")  //nolint:forbidigo
		_ = os.Remove(t.prefix + ".out") //nolint:forbidigo
	})
	return nil
}

type fifoConn struct {
	reader *os.File //nolint:forbidigo
	writer *os.File //nolint:forbidigo
}

func (c *fifoConn) Read(p []byte) (int, error) {
	return c.reader.Read(p) //nolint:forbidigo
}

func (c *fifoConn) Write(p []byte) (int, error) {
	return c.writer.Write(p) //nolint:forbidigo
}

func (c *fifoConn) Close() error {
	err1 := c.reader.Close() //nolint:forbidigo
	err2 := c.writer.Close() //nolint:forbidigo
	if err1 != nil {
		return err1
	}
	return err2
}
