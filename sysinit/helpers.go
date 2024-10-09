package sysinit

import (
	"log"
	"os"
	"path/filepath"

	"sideboot/exec"

	"golang.org/x/sys/unix"
)

const (
	ExecAll     = 0o777
	ReadExecAll = 0o555
)

type Exec []string

func (e Exec) Run() exec.Status {
	cmd := exec.NewCmd(e[0], e[1:]...)
	status := <-cmd.Start()

	for _, line := range status.Stderr {
		log.Printf("[%d] %s: %s", status.PID, e[0], line)
	}

	return status
}

func (e Exec) Lines() (output []string) {
	status := e.Run()
	if status.Exit == 0 {
		output = status.Stdout
	}

	return
}

func (e Exec) Line(index int) string {
	output := e.Lines()
	if len(output) <= index {
		return ""
	}

	return output[index]
}

type Dir struct {
	Path string
	Mode os.FileMode
}

func (d Dir) Run() error {
	return os.MkdirAll(filepath.Join("/", d.Path), d.Mode)
}

type Symlink struct {
	Path   string
	Target string
}

func (s Symlink) Run() error {
	os.Remove(s.Path)
	return os.Symlink(s.Target, s.Path)
}

type Special struct {
	Path string
	Mode uint32
	Dev  int
}

func (d Special) Run() error {
	os.Remove(d.Path)
	return unix.Mknod(d.Path, d.Mode, d.Dev)
}

type Mount struct {
	Source string
	Target string
	Type   string
	Flags  uintptr
	Opts   string
}

func (m Mount) Run() error {
	return unix.Mount(m.Source, m.Target, m.Type, m.Flags, m.Opts)
}

func Bind(source string, target string) Mount {
	return Mount{
		Source: filepath.Join("/", source),
		Target: filepath.Join("/", target),
		Type:   "none",
		Flags:  unix.MS_BIND,
		Opts:   "",
	}
}

func MountProc(target string) Mount {
	return Mount{
		Source: "proc",
		Target: filepath.Join("/", target),
		Type:   "proc",
		Flags:  0,
		Opts:   "",
	}
}

func MountDev(target string) Mount {
	return Mount{
		Source: "devtmpfs",
		Target: filepath.Join("/", target),
		Type:   "devtmpfs",
		Flags:  0,
		Opts:   "",
	}
}

func MountDevPts(target string) Mount {
	return Mount{
		Source: "devpts",
		Target: filepath.Join("/", target),
		Type:   "devpts",
		Flags:  0,
		Opts:   "newinstance,ptmxmode=666,gid=5,mode=620",
	}
}

func MountTmp(target string) Mount {
	return Mount{
		Source: "tmpfs",
		Target: filepath.Join("/", target),
		Type:   "tmpfs",
		Flags:  0,
		Opts:   "",
	}
}

func MountSys(target string) Mount {
	return Mount{
		Source: "sysfs",
		Target: filepath.Join("/", target),
		Type:   "sysfs",
		Flags:  0,
		Opts:   "",
	}
}
