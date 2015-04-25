package splice

import (
	"errors"
	"io"
	"os"
	"sync"
	"syscall"
)

const (
	SPLICE_F_MORE     = 0x01
	SPLICE_F_NONBLOCK = 0x02
	SPLICE_F_MOVE     = 0x03
)

var (
	ErrNoFD error = errors.New("The requested stream does not have Fd() method")
)

var (
	pipeFct   = syscall.Pipe
	spliceFct = syscall.Splice
)

type Splice struct {
	sync.Mutex
	pipe       []int
	bufferSize int
	flags      int
}

type Filer interface {
	File() (*os.File, error)
}

type FDer interface {
	Fd() uintptr
}

func (s *Splice) SetBufferSize(size int) {
	s.bufferSize = size
}

func (s *Splice) SetFlags(flags int) {
	s.flags = flags
}

func (s *Splice) Copy(dst io.Writer, src io.Reader) (n int64, err error) {
	s.Lock()
	defer s.Unlock()

	var (
		written int64
		srcFd   int
		dstFd   int
	)

	if f, ok := src.(FDer); !ok {
		if f, ok := src.(Filer); !ok {
			return -1, ErrNoFD
		} else {
			if f, err := f.File(); err != nil {
				return -1, err
			} else {
				srcFd = int(f.Fd())
			}
		}
	} else {
		srcFd = int(f.Fd())
	}

	if t, ok := dst.(FDer); !ok {
		if t, ok := dst.(Filer); !ok {
			return -1, ErrNoFD
		} else {
			if t, err := t.File(); err != nil {
				return -1, err
			} else {
				dstFd = int(t.Fd())
			}
		}
	} else {
		dstFd = int(t.Fd())
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

func Copy(dst io.Writer, src io.Reader) (n int64, err error) {
	s, err := NewSplice()
	if err != nil {
		return -1, err
	}
	defer s.Close()
	return s.Copy(dst, src)
}

func (s *Splice) Close() error {
	if s.pipe[0] != 0 {
		os.NewFile(uintptr(s.pipe[0]), "").Close()
	}
	if s.pipe[1] != 0 {
		os.NewFile(uintptr(s.pipe[1]), "").Close()
	}
	return nil
}

func NewSplice() (*Splice, error) {
	pipe := []int{0, 0}
	if err := pipeFct(pipe); err != nil {
		return nil, err
	}
	return &Splice{
		pipe:       pipe,
		bufferSize: 32 * 1024,
		flags:      SPLICE_F_MOVE | SPLICE_F_MORE,
	}, nil
}
