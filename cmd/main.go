package main

import (
	"log"
	"os"
	"strings"
	"syscall"

	"sideboot/sysinit"

	"github.com/kballard/go-shellquote"
)

const (
	kernelOption    = "sideboot.kernel"
	shellOption     = "sideboot.shell"
	ramdiskOption   = "sideboot.ramdisk"
	cmdlineOption   = "sideboot.cmdline"
	partitionOption = "sideboot.partition"
)

func resetBootOptions() {
	sysinit.Args[partitionOption] = ""
	sysinit.Args[shellOption] = ""
	sysinit.Args[kernelOption] = ""
	sysinit.Args[ramdiskOption] = ""
	sysinit.Args[cmdlineOption] = "console=tty1 loglevel=4"
}

func tryBoot() bool {
	if sysinit.Args[partitionOption] == "" {
		return false
	}

	filename := (sysinit.Exec{"/bin/findfs", sysinit.Args[partitionOption]}).Line(0)
	if filename == "" {
		return false
	}

	sysinit.Dir{Path: "tmp/boot", Mode: 0x777}.Run()
	err := sysinit.Mount{Type: "ext2", Flags: syscall.MS_RDONLY, Source: filename, Target: "/tmp/boot"}.Run()
	defer func() {
		os.Chdir("/")
		syscall.Unmount("/tmp/boot", 0)
	}()

	if err != nil {
		log.Print(filename, err)
		return false
	}

	cfg := strings.ReplaceAll(sysinit.ReadFile("/tmp/boot/sideboot.cmdline"), "\n", " ")

	cfgArgs, err := shellquote.Split(cfg)
	if err != nil {
		log.Print("cfgArgs", filename, err)
		return false
	}

	bootPartition := sysinit.Args[partitionOption]
	sysinit.ParseArgs(cfgArgs)

	if sysinit.AsInit() && sysinit.Args[shellOption] == "1" {
		log.Print("welcome to sideboot shell")
		return false
	}

	if bootPartition != sysinit.Args[partitionOption] {
		return tryBoot()
	}

	os.Chdir("/tmp/boot")

	kexec := sysinit.Exec{"/libexec/kexec", "--command-line", sysinit.Args[cmdlineOption]}

	if sysinit.Args[kernelOption] == "" || !sysinit.FileExist(sysinit.Args[kernelOption]) {
		return false
	}

	if sysinit.Args[ramdiskOption] != "" {
		if !sysinit.FileExist(sysinit.Args[ramdiskOption]) {
			log.Print("ramdisk", sysinit.Args[ramdiskOption], "not found")
			return false
		}

		kexec = append(kexec, "--initrd", sysinit.Args[ramdiskOption])
	}

	kexec = append(kexec, "--load", sysinit.Args[kernelOption])
	status := kexec.Run()
	if status.Exit == 0 {
		status = (sysinit.Exec{"/libexec/kexec", "--exec"}).Run()
		if status.Exit == 0 {
			return true
		}
	}

	log.Print("kexec", status.Exit)
	os.Chdir("/")

	return false
}

func main() {
	resetBootOptions()
	sysinit.Init()

	if !tryBoot() {
		os.Chdir("/")
		sysinit.DebugShell()
	}

	sysinit.Exit()
}
