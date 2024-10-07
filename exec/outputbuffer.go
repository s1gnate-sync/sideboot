package exec

import (
	"bufio"
	"bytes"
	"fmt"
	"sync"
)

type OutputBuffer struct {
	buf   *bytes.Buffer
	lines []string
	*sync.Mutex
}

func NewOutputBuffer() *OutputBuffer {
	out := &OutputBuffer{
		buf:   &bytes.Buffer{},
		lines: []string{},
		Mutex: &sync.Mutex{},
	}
	return out
}

func (rw *OutputBuffer) Write(p []byte) (n int, err error) {
	rw.Lock()
	n, err = rw.buf.Write(p)
	rw.Unlock()
	return
}

func (rw *OutputBuffer) Lines() []string {
	rw.Lock()

	s := bufio.NewScanner(rw.buf)
	for s.Scan() {
		rw.lines = append(rw.lines, s.Text())
	}
	rw.Unlock()
	return rw.lines
}

const (
	DEFAULT_LINE_BUFFER_SIZE = 16384

	DEFAULT_STREAM_CHAN_SIZE = 1000
)

type ErrLineBufferOverflow struct {
	Line       string
	BufferSize int
	BufferFree int
}

func (e ErrLineBufferOverflow) Error() string {
	return fmt.Sprintf("line does not contain newline and is %d bytes too long to buffer (buffer size: %d)",
		len(e.Line)-e.BufferSize, e.BufferSize)
}
