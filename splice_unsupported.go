// +build !linux

package gosplice

import (
	"errors"
	"io"
	"os"
)

// Common errors.
var ErrUnsupportedOS = errors.New("splice is supported only on linux")

// Splice represent the needed data for a splice.
type Splice struct{}

// Filer is an interface to get the File out of an object.
type Filer interface {
	File() (*os.File, error)
}

// FDer is an interface to get the FD out of an object.
type FDer interface {
	Fd() uintptr
}

// SetBufferSize sets the buffer size.
func (s *Splice) SetBufferSize(size int) {}

// SetFlags sets the Splice flags.
func (s *Splice) SetFlags(flags int) {
}

// Copy effectively perform the splice operation.
func (s *Splice) Copy(dst io.Writer, src io.Reader) (n int64, err error) {
	return 0, ErrUnsupportedOS
}

// Copy is a helper that instantiates a new Splice and perform the Copy.
func Copy(dst io.Writer, src io.Reader) (n int64, err error) {
	return 0, ErrUnsupportedOS
}

// Close terminates the splice.
func (s *Splice) Close() error {
	return ErrUnsupportedOS
}

// NewSplice instantiates a new Splice object.
func NewSplice() (*Splice, error) {
	return nil, ErrUnsupportedOS
}
