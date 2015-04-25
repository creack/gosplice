// +build linux

package gosplice

import (
	"errors"
	"io"
	"os"
	"sync"
	"syscall"
)

// Splice options consts.
const (
	SpliceFMore     = 0x01
	SpliceFNonblock = 0x02
	SpliceFMove     = 0x03
)

// Common error
var ErrNoFD = errors.New("The requested stream does not have Fd() method")

var (
	pipeFct   = syscall.Pipe
	spliceFct = syscall.Splice
)

// Splice represent the needed data for a splice.
type Splice struct {
	sync.Mutex
	pipe       []int
	bufferSize int
	flags      int
}

// Filer is an interface to get the File out of an object.
type Filer interface {
	File() (*os.File, error)
}

// FDer is an interface to get the FD out of an object.
type FDer interface {
	Fd() uintptr
}

// SetBufferSize sets the buffer size.
func (s *Splice) SetBufferSize(size int) {
	s.bufferSize = size
}

// SetFlags sets the Splice flags.
func (s *Splice) SetFlags(flags int) {
	s.flags = flags
}

// Copy effectively perform the splice operation.
func (s *Splice) Copy(dst io.Writer, src io.Reader) (n int64, err error) {
	s.Lock()
	defer s.Unlock()

	var (
		written int64
		srcFd   int
		dstFd   int
	)

	if f, ok := src.(FDer); !ok {
		filer, ok := src.(Filer)
		if !ok {
			return -1, ErrNoFD
		}
		f, err := filer.File()
		if err != nil {
			return -1, err
		}
		srcFd = int(f.Fd())
	} else {
		srcFd = int(f.Fd())
	}

	if fder, ok := dst.(FDer); ok {
		dstFd = int(fder.Fd())
	} else {
		filer, ok := dst.(Filer)
		if !ok {
			return -1, ErrNoFD
		}
		f, err := filer.File()
		if err != nil {
			return -1, err
		}
		dstFd = int(f.Fd())
	}

	for {
		w, err := spliceFct(srcFd, nil, s.pipe[1], nil, s.bufferSize, s.flags)
		if err != nil {
			return written, err
		}
		if w == 0 {
			break
		}
		w, err = spliceFct(s.pipe[0], nil, dstFd, nil, int(w), s.flags)
		if err != nil {
			return written + int64(w), err
		}
		written += int64(w)
	}
	return written, nil
}

// Copy is a helper that instantiates a new Splice and perform the Copy.
func Copy(dst io.Writer, src io.Reader) (n int64, err error) {
	s, err := NewSplice()
	if err != nil {
		return -1, err
	}
	defer s.Close()
	return s.Copy(dst, src)
}

// Close terminates the splice.
func (s *Splice) Close() error {
	if s.pipe[0] != 0 {
		os.NewFile(uintptr(s.pipe[0]), "").Close()
	}
	if s.pipe[1] != 0 {
		os.NewFile(uintptr(s.pipe[1]), "").Close()
	}
	return nil
}

// NewSplice instantiates a new Splice object.
func NewSplice() (*Splice, error) {
	pipe := []int{0, 0}
	if err := pipeFct(pipe); err != nil {
		return nil, err
	}
	return &Splice{
		pipe:       pipe,
		bufferSize: 32 * 1024,
		flags:      SpliceFMove | SpliceFMore,
	}, nil
}
