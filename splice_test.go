package splice

import (
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
	"testing"
)

var listenSpliceFct = listenSplice

func listenPong(port int, ready chan bool) error {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	defer l.Close()
	ready <- true
	rw, err := l.Accept()
	if err != nil {
		return err
	}
	if _, err := io.Copy(rw, rw); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return nil
}

func listenSplice(listenPort, dstPort int, ready chan bool) error {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", listenPort))
	if err != nil {
		return err
	}
	defer l.Close()
	ready <- true

	rw, err := l.Accept()
	if err != nil {
		return err
	}
	backend, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", dstPort))
	if err != nil {
		rw.Close()
		return err
	}
	ret := make(chan error, 2)
	go func() {
		defer rw.Close()
		if _, err := Copy(rw, backend); err != nil {
			if err == io.EOF {
				ret <- nil
				return
			}
			ret <- err
		}
	}()
	go func() {
		defer backend.Close()
		defer rw.Close()
		if _, err := Copy(backend, rw); err != nil {
			if err == io.EOF {
				ret <- nil
				return
			}
			ret <- err
		}
	}()
	err1 := <-ret
	err2 := <-ret
	if err1 != nil && err2 != nil {
		return fmt.Errorf("Error copy1: %s, Error copy2: %s", err1, err2)
	} else if err1 != nil {
		return err1
	} else if err2 != nil {
		return err2
	}
	return nil
}

func TestNewSplice(t *testing.T) {
	s, err := NewSplice()
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("Failed to instanciate Splice struct")
	}

	pipeFct = pipeFctFail
	if _, err := NewSplice(); err == nil {
		t.Fatal("Pipe excpected to fail but didn't")
	}
	pipeFct = syscall.Pipe
}

func TestSetFlags(t *testing.T) {
	s, err := NewSplice()
	if err != nil {
		t.Fatal(err)
	}
	s.SetFlags(SPLICE_F_MORE)
	if s.flags != SPLICE_F_MORE {
		t.Fatalf("SetFlags failed, expected flag: %d, found: %d", SPLICE_F_MORE, s.flags)
	}
	s.SetFlags(SPLICE_F_MORE | SPLICE_F_MOVE)
	if s.flags != SPLICE_F_MORE|SPLICE_F_MOVE {
		t.Fatalf("SetFlags failed, expected flag: %d, found: %d", SPLICE_F_MORE|SPLICE_F_MOVE, s.flags)
	}
	s.SetFlags(0)
	if s.flags != 0 {
		t.Fatalf("SetFlags failed, expected flag: %d, found: %d", 0, s.flags)
	}
}

func TestSetBufferSize(t *testing.T) {
	s, err := NewSplice()
	if err != nil {
		t.Fatal(err)
	}
	s.SetBufferSize(32 * 1024)
	if s.bufferSize != 32*1024 {
		t.Fatalf("SetBufferSize failed, expected size: %d, found: %d", 32*1024, s.bufferSize)
	}
	s.SetBufferSize(0)
	if s.bufferSize != 0 {
		t.Fatalf("SetBufferSize failed, expected size: %d, found: %d", 0, s.bufferSize)
	}
}

func testCopy(listenPort, dstPort int) error {
	ready := make(chan bool, 2)
	ret := make(chan error, 3)
	go func() {
		if err := listenPong(dstPort, ready); err != nil {
			ret <- err
			return
		}
		ret <- nil
	}()
	go func() {
		if err := listenSpliceFct(listenPort, dstPort, ready); err != nil {
			ret <- err
			return
		}
		ret <- nil
	}()

	go func() {
		client, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", listenPort))
		if err != nil {
			ret <- err
			return
		}
		count := 15
		input := "hello world!"
		output := input
		for i := 0; i < count; i++ {
			if _, err := client.Write([]byte(input)); err != nil {
				ret <- err
				return
			}
			buf := make([]byte, len(input))
			if _, err := client.Read(buf); err != nil {
				if err != io.EOF {
					ret <- err
				} else {
					ret <- nil
				}
				return
			}
			if string(buf) != output {
				ret <- fmt.Errorf("Unexpected output (%d). Expected [%s], received [%s]", i, output, string(buf))
				return
			}
		}
		ret <- nil
	}()

	select {
	case <-ready:
	case r := <-ret:
		if r != nil {
			return r
		}
	}
	select {
	case <-ready:
	case r := <-ret:
		if r != nil {
			return r
		}
	}

	return <-ret
}

func TestCopyHelper(t *testing.T) {
	if err := testCopy(54931, 54932); err != nil {
		t.Fatal(err)
	}
	pipeFct = pipeFctFail
	if err := testCopy(54933, 54934); err == nil {
		t.Fatal("Excpected Copy to fail but didn't")
	}
	pipeFct = syscall.Pipe

	spliceFct = spliceFctFail
	if err := testCopy(54935, 54936); err == nil {
		t.Fatal("Excpected Copy to fail but didn't")
	}
	spliceFct = syscall.Splice

	// spliceFct = spliceFctFail2
	// if err := testCopy(54937, 54938); err == nil {
	// 	t.Fatal("Excpected Copy to fail but didn't")
	// }
	// spliceFct = syscall.Splice

}

func TestCopy(t *testing.T) {
	s, err := NewSplice()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Copy(nil, nil); err == nil {
		t.Fatal("Excpected Copy to fail but didn't")
	}

	if _, err := s.Copy(nil, &fder{}); err == nil {
		t.Fatal("Excpected Copy to fail but didn't")
	}

	if _, err := s.Copy(&fder{}, &fder{}); err == nil {
		t.Fatal("Excpected Copy to fail but didn't")
	}

	if _, err := s.Copy(nil, &filer{}); err == nil {
		t.Fatal("Excpected Copy to fail but didn't")
	}

	if _, err := s.Copy(&filer{}, &fder{}); err == nil {
		t.Fatal("Excpected Copy to fail but didn't")
	}
}

func BenchmarkCopy(b *testing.B) {
	b.Skip("Unimplemented")

	b.StopTimer()

	println("HELLO")
	dstPort := 54772
	listenPort := 54773

	ready := make(chan bool, 2)
	ret := make(chan error, 3)
	go func() {
		if err := listenPong(dstPort, ready); err != nil {
			ret <- err
			return
		}
		ret <- nil
	}()
	go func() {
		if err := listenSpliceFct(listenPort, dstPort, ready); err != nil {
			ret <- err
			return
		}
		ret <- nil
	}()

	b.StartTimer()
	for i := 0; i < 1; i++ {
		client, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", listenPort))
		if err != nil {
			ret <- err
			return
		}
		count := 15
		input := "hello world!"
		output := input
		for i := 0; i < count; i++ {
			if _, err := client.Write([]byte(input)); err != nil {
				ret <- err
				return
			}
			buf := make([]byte, len(input))
			if _, err := client.Read(buf); err != nil {
				if err != io.EOF {
					ret <- err
					return
				}
			}
			if string(buf) != output {
				ret <- fmt.Errorf("Unexpected output (%d). Expected [%s], received [%s]", i, output, string(buf))
				return
			}
		}
		ret <- nil
	}
}

// Helpers

type fder struct {
	io.Writer
	io.Reader
}

func (fder) Fd() uintptr {
	return 255
}

type filer struct {
	io.Writer
	io.Reader
}

func (filer) File() (*os.File, error) {
	return nil, fmt.Errorf("Fail")
}

func spliceFctFail(rfd int, roff *int64, wfd int, woff *int64, len int, flags int) (n int64, err error) {
	return -1, fmt.Errorf("Fail")
}

var spliceFctCount = 0

func spliceFctFail2(rfd int, roff *int64, wfd int, woff *int64, len int, flags int) (n int64, err error) {
	if spliceFctCount == 1 {
		spliceFctCount = 0
		return -1, fmt.Errorf("Fail")
	}
	spliceFctCount++
	return syscall.Splice(rfd, roff, wfd, woff, len, flags)
}

func pipeFctFail(fds []int) error {
	return fmt.Errorf("Fail")
}
