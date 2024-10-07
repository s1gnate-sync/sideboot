package exec

import "bytes"

type OutputStream struct {
	streamChan chan string
	bufSize    int
	buf        []byte
	lastChar   int
}

func NewOutputStream(streamChan chan string) *OutputStream {
	out := &OutputStream{
		streamChan: streamChan,

		bufSize:  DEFAULT_LINE_BUFFER_SIZE,
		buf:      make([]byte, DEFAULT_LINE_BUFFER_SIZE),
		lastChar: 0,
	}
	return out
}

func (rw *OutputStream) Write(p []byte) (n int, err error) {
	n = len(p)
	firstChar := 0

	for {

		newlineOffset := bytes.IndexByte(p[firstChar:], '\n')
		if newlineOffset < 0 {
			break
		}

		lastChar := firstChar + newlineOffset
		if newlineOffset > 0 && p[newlineOffset-1] == '\r' {
			lastChar -= 1
		}

		var line string
		if rw.lastChar > 0 {
			line = string(rw.buf[0:rw.lastChar])
			rw.lastChar = 0
		}
		line += string(p[firstChar:lastChar])
		rw.streamChan <- line

		firstChar += newlineOffset + 1
	}

	if firstChar < n {
		remain := len(p[firstChar:])
		bufFree := len(rw.buf[rw.lastChar:])
		if remain > bufFree {
			var line string
			if rw.lastChar > 0 {
				line = string(rw.buf[0:rw.lastChar])
			}
			line += string(p[firstChar:])
			err = ErrLineBufferOverflow{
				Line:       line,
				BufferSize: rw.bufSize,
				BufferFree: bufFree,
			}
			n = firstChar
			return
		}
		copy(rw.buf[rw.lastChar:], p[firstChar:])
		rw.lastChar += remain
	}

	return
}

func (rw *OutputStream) Lines() <-chan string {
	return rw.streamChan
}

func (rw *OutputStream) SetLineBufferSize(n int) {
	rw.bufSize = n
	rw.buf = make([]byte, rw.bufSize)
}

func (rw *OutputStream) Flush() {
	if rw.lastChar > 0 {
		line := string(rw.buf[0:rw.lastChar])
		rw.streamChan <- line
	}
}
