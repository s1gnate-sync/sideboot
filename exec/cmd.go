package exec

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Cmd struct {
	Name string

	Args []string

	Env []string

	Dir string

	Stdout chan string

	Stderr chan string

	*sync.Mutex
	started         bool
	stopped         bool
	done            bool
	final           bool
	startTime       time.Time
	stdoutBuf       *OutputBuffer
	stderrBuf       *OutputBuffer
	stdoutStream    *OutputStream
	stderrStream    *OutputStream
	status          Status
	statusChan      chan Status
	doneChan        chan struct{}
	beforeExecFuncs []func(cmd *exec.Cmd)
}

var ErrNotStarted = errors.New("command not running")

type Status struct {
	Cmd      string
	PID      int
	Complete bool
	Exit     int
	Error    error
	StartTs  int64
	StopTs   int64
	Runtime  float64
	Stdout   []string
	Stderr   []string
}

func NewCmd(name string, args ...string) *Cmd {
	return NewCmdOptions(Options{Buffered: true}, name, args...)
}

type Options struct {
	Buffered bool

	CombinedOutput bool

	Streaming bool

	BeforeExec func(cmd *exec.Cmd)

	LineBufferSize uint
}

func NewCmdOptions(options Options, name string, args ...string) *Cmd {
	c := &Cmd{
		Name: name,
		Args: args,

		Mutex: &sync.Mutex{},
		status: Status{
			Cmd:      name,
			PID:      0,
			Complete: false,
			Exit:     -1,
			Error:    nil,
			Runtime:  0,
		},
		doneChan: make(chan struct{}),
	}

	if options.LineBufferSize == 0 {
		options.LineBufferSize = DEFAULT_LINE_BUFFER_SIZE
	}

	if options.Buffered {
		c.stdoutBuf = NewOutputBuffer()
		c.stderrBuf = NewOutputBuffer()
	}

	if options.CombinedOutput {
		c.stdoutBuf = NewOutputBuffer()
		c.stderrBuf = nil
	}

	if options.Streaming {
		c.Stdout = make(chan string, DEFAULT_STREAM_CHAN_SIZE)
		c.stdoutStream = NewOutputStream(c.Stdout)
		c.stdoutStream.SetLineBufferSize(int(options.LineBufferSize))

		c.Stderr = make(chan string, DEFAULT_STREAM_CHAN_SIZE)
		c.stderrStream = NewOutputStream(c.Stderr)
		c.stderrStream.SetLineBufferSize(int(options.LineBufferSize))
	}

	if options.BeforeExec != nil {
		c.beforeExecFuncs = []func(cmd *exec.Cmd){options.BeforeExec}
	}

	return c
}

func (c *Cmd) Clone() *Cmd {
	clone := NewCmdOptions(
		Options{
			Buffered:       c.stdoutBuf != nil,
			CombinedOutput: c.stdoutBuf != nil,
			Streaming:      c.stdoutStream != nil,
		},
		c.Name,
		c.Args...,
	)
	clone.Dir = c.Dir
	clone.Env = c.Env

	if len(c.beforeExecFuncs) > 0 {
		clone.beforeExecFuncs = make([]func(cmd *exec.Cmd), len(c.beforeExecFuncs))
		copy(clone.beforeExecFuncs, c.beforeExecFuncs)
	}

	return clone
}

func (c *Cmd) Start() <-chan Status {
	return c.StartWithStdin(nil)
}

func (c *Cmd) StartWithStdin(in io.Reader) <-chan Status {
	c.Lock()
	defer c.Unlock()

	if c.statusChan != nil {
		return c.statusChan
	}
	c.statusChan = make(chan Status, 1)

	go c.run(in)
	return c.statusChan
}

func (c *Cmd) Stop() error {
	c.Lock()
	defer c.Unlock()

	if c.stopped {
		return nil
	}
	c.stopped = true

	if c.statusChan == nil || !c.started {
		return ErrNotStarted
	}

	if c.done {
		return nil
	}

	return terminateProcess(c.status.PID)
}

func (c *Cmd) Status() Status {
	c.Lock()
	defer c.Unlock()

	if c.statusChan == nil || !c.started {
		return c.status
	}

	if c.done {
		if !c.final {
			if c.stdoutBuf != nil {
				c.status.Stdout = c.stdoutBuf.Lines()
				c.stdoutBuf = nil

			}
			if c.stderrBuf != nil {
				c.status.Stderr = c.stderrBuf.Lines()
				c.stderrBuf = nil
			}
			c.final = true
		}
	} else {
		c.status.Runtime = time.Since(c.startTime).Seconds()
		if c.stdoutBuf != nil {
			c.status.Stdout = c.stdoutBuf.Lines()
		}
		if c.stderrBuf != nil {
			c.status.Stderr = c.stderrBuf.Lines()
		}
	}

	return c.status
}

func (c *Cmd) Done() <-chan struct{} {
	return c.doneChan
}

func (c *Cmd) run(in io.Reader) {
	defer func() {
		c.statusChan <- c.Status()
		close(c.doneChan)
	}()

	cmd := exec.Command(c.Name, c.Args...)
	if in != nil {
		cmd.Stdin = in
	}

	setProcessGroupID(cmd)

	switch {
	case c.stdoutBuf != nil && c.stderrBuf != nil && c.stdoutStream != nil:
		cmd.Stdout = io.MultiWriter(c.stdoutStream, c.stdoutBuf)
		cmd.Stderr = io.MultiWriter(c.stderrStream, c.stderrBuf)
	case c.stdoutBuf != nil && c.stderrBuf == nil && c.stdoutStream != nil:
		cmd.Stdout = io.MultiWriter(c.stdoutStream, c.stdoutBuf)
		cmd.Stderr = io.MultiWriter(c.stderrStream, c.stdoutBuf)
	case c.stdoutBuf != nil && c.stderrBuf != nil:
		cmd.Stdout = c.stdoutBuf
		cmd.Stderr = c.stderrBuf
	case c.stdoutBuf != nil && c.stderrBuf == nil:
		cmd.Stdout = c.stdoutBuf
		cmd.Stderr = c.stdoutBuf
	case c.stdoutStream != nil:
		cmd.Stdout = c.stdoutStream
		cmd.Stderr = c.stderrStream
	default:
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	if c.stdoutStream != nil {
		defer func() {
			c.stdoutStream.Flush()
			c.stderrStream.Flush()

			close(c.Stdout)
			close(c.Stderr)
		}()
	}

	cmd.Env = c.Env
	cmd.Dir = c.Dir

	for _, f := range c.beforeExecFuncs {
		f(cmd)

		c.Lock()
		stopped := c.stopped
		c.Unlock()
		if stopped {
			return
		}
	}

	now := time.Now()
	if err := cmd.Start(); err != nil {
		c.Lock()
		c.status.Error = err
		c.status.StartTs = now.UnixNano()
		c.status.StopTs = time.Now().UnixNano()
		c.done = true
		c.Unlock()
		return
	}

	c.Lock()
	c.startTime = now
	c.status.PID = cmd.Process.Pid
	c.status.StartTs = now.UnixNano()
	c.started = true
	c.Unlock()

	err := cmd.Wait()
	now = time.Now()

	exitCode := 0
	signaled := false
	if err != nil && fmt.Sprintf("%T", err) == "*exec.ExitError" {

		exiterr := err.(*exec.ExitError)
		err = nil
		if waitStatus, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			exitCode = waitStatus.ExitStatus()
			if waitStatus.Signaled() {
				signaled = true
				err = errors.New(exiterr.Error())
			}
		}
	}

	c.Lock()
	if !c.stopped && !signaled {
		c.status.Complete = true
	}
	c.status.Runtime = now.Sub(c.startTime).Seconds()
	c.status.StopTs = now.UnixNano()
	c.status.Exit = exitCode
	c.status.Error = err
	c.done = true
	c.Unlock()
}

func terminateProcess(pid int) error {
	return syscall.Kill(-pid, syscall.SIGTERM)
}

func setProcessGroupID(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
