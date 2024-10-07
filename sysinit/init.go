package sysinit

import (
	"fmt"
	"log"
	"os"
	osexec "os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

var (
	Verbose = false
	Quiet   = false
	Args    = make(map[string]string)
	inited  = false
)

func AsInit() bool {
	return !inited
}

func init() {
	Args["debug"] = ""
	Args["loglevel"] = "4"

	unix.Umask(0)
	os.Chdir("/")

	inited = os.Getenv("SIDEBOOT") == "1"

	os.Clearenv()

	for key, value := range map[string]string{
		"PATH":       "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"LANG":       "C.UTF-8",
		"LC_COLLATE": "C",
		"CHARSET":    "UTF-8",
		"TERM":       "xterm-256color",
		"HOME":       "/tmp",
		"SIDEBOOT":   "1",
	} {
		os.Setenv(key, value)
	}
}

func Init() {
	if inited {
		ParseArgs(os.Args[1:])
		return
	}

	unix.Syscall(unix.SYS_SYSLOG, unix.SYSLOG_ACTION_CONSOLE_OFF, 0, 0)
	CreateDirs()
	MountFilesystems()

	ParseArgs([]string{})
	if Args["debug"] == "1" {
		Quiet = false
		Verbose = true
	}

	SetLog()
	InstallBusybox()
	MakeDevs()
}

func MakeDevs() {
	if err := (Special{Path: "/dev/tty", Mode: unix.S_IFCHR | 0o666, Dev: 0x0500}).Run(); err != nil {
		log.Print("tty: ", err)
	}

	if err := (Symlink{Target: "dev/pts/ptmx", Path: "dev/tmpx"}).Run(); err != nil {
		log.Print("ptmx: ", err)
	}
}

func SetLog() {
	if !Quiet || Verbose {
		log.SetOutput(os.Stderr)
	}

	level := 0
	if _, err := fmt.Sscanf(Args["loglevel"], "%d", &level); err != nil {
		level = 4
	}

	if Verbose {
		level = 7
	}

	unix.Syscall(unix.SYS_SYSLOG, unix.SYSLOG_ACTION_CONSOLE_LEVEL, 0, uintptr(level))
}

func MountFilesystems() {
	for _, mnt := range []Mount{
		MountDev("dev"), MountProc("proc"), MountSys("sys"), MountTmp("tmp"),
	} {
		if err := mnt.Run(); err != nil {
			log.Print("mnt: ", mnt.Type, err)
		}
	}

	Dir{Path: "dev/pts", Mode: 0x755}.Run()
	if err := MountDevPts("dev/pts").Run(); err != nil {
		log.Print("mnt: devpts ", err)
	}
}

func InstallBusybox() {
	for _, line := range (Exec{"/libexec/busybox", "--list"}).Lines() {
		if err := (Symlink{Target: "/libexec/busybox", Path: filepath.Join("bin", line)}).Run(); err != nil {
			log.Print("bb: ", line, err)
		}
	}
}

func CreateDirs() {
	for _, dir := range []string{
		"bin", "dev", "proc", "root", "sbin", "lib", "libexec", "sys", "tmp", "usr/bin", "usr/sbin",
	} {
		if err := (Dir{Path: dir, Mode: 0o755}).Run(); err != nil {
			log.Print("dir: ", dir, err)
		}
	}
}

func DebugShell() {
	cmd := osexec.Command("/libexec/busybox", "ash", "-l")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
	os.Exit(cmd.ProcessState.ExitCode())
}

func Exit() {
	WaitOrphans()
	unix.Sync()
	os.Exit(0)
}

func WaitOrphans() uint {
	var numReaped uint
	for {
		var (
			s unix.WaitStatus
			r unix.Rusage
		)
		p, _ := unix.Wait4(-1, &s, 0, &r)
		if p == -1 {
			break
		}
		numReaped++
	}
	return numReaped
}
