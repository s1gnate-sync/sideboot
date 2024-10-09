package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"sideboot/sysinit"

	"github.com/kballard/go-shellquote"
)

const (
	kernelOption    = "sideboot.kernel"
	shellOption     = "sideboot.shell"
	ramdiskOption   = "sideboot.ramdisk"
	cmdlineOption   = "sideboot.cmdline"
	partitionOption = "sideboot.partition"
	configOption    = "sideboot.config"
)

func resetBootOptions() {
	sysinit.Args[partitionOption] = ""
	sysinit.Args[shellOption] = ""
	sysinit.Args[kernelOption] = ""
	sysinit.Args[ramdiskOption] = ""
	sysinit.Args[configOption] = ""
	sysinit.Args[cmdlineOption] = "console=tty1 loglevel=4"
}

var bootMsg string

func tryBoot() bool {
	bootMsg = ""

	if !wait() {
		bootMsg = "user gesture interrupted boot"
		return false
	}

	if sysinit.Args[partitionOption] == "" {
		bootMsg = "no boot partition has been specified"
		return false
	}

	filename := (sysinit.Exec{"/bin/findfs", sysinit.Args[partitionOption]}).Line(0)
	if filename == "" {
		bootMsg = fmt.Sprintf("boot partition %s doesn't point to device", sysinit.Args[partitionOption])

		return false
	}

	sysinit.Dir{Path: "tmp/boot", Mode: 0x777}.Run()
	err := sysinit.Mount{Type: "ext2", Flags: syscall.MS_RDONLY, Source: filename, Target: "/tmp/boot"}.Run()
	defer func() {
		os.Chdir("/")
		syscall.Unmount("/tmp/boot", 0)
	}()

	if err != nil {
		bootMsg = fmt.Sprintf("error on mount %s: %s", filename, err)
		return false
	}

	cfg := ""
	if sysinit.Args[configOption] != "" {
		cfg = strings.ReplaceAll(sysinit.ReadFile(filepath.Join("/tmp/boot/", sysinit.Args[configOption])), "\n", " ")
	}

	if cfg == "" {
		cfg = strings.ReplaceAll(sysinit.ReadFile(filepath.Join("/tmp/boot/sideboot.cfg")), "\n", " ")
	}

	cfgArgs, err := shellquote.Split(cfg)
	if err != nil {
		bootMsg = "commmandline to next kernel contains garbage"
		return false
	}

	if !sysinit.AsInit() {
		for _, arg := range os.Args[1:] {
			cfgArgs = append(cfgArgs, "sideboot."+arg)
		}
	}

	bootPartition := sysinit.Args[partitionOption]
	sysinit.ParseArgs(cfgArgs)

	if sysinit.AsInit() && sysinit.Args[shellOption] == "1" {
		bootMsg = "not booting because default action is set to debug shell"
		return false
	}

	if bootPartition != sysinit.Args[partitionOption] {
		os.Chdir("/")
		syscall.Unmount("/tmp/boot", 0)
		return tryBoot()
	}

	os.Chdir("/tmp/boot")

	kexec := sysinit.Exec{"/libexec/kexec", "--command-line", sysinit.Args[cmdlineOption]}

	if sysinit.Args[kernelOption] == "" || !sysinit.FileExist(sysinit.Args[kernelOption]) {
		bootMsg = fmt.Sprintf("boot requires kernel to be set to existing file on device %s", bootPartition)
		return false
	}

	if sysinit.Args[ramdiskOption] != "" {
		if !sysinit.FileExist(sysinit.Args[ramdiskOption]) {
			bootMsg = fmt.Sprintf("ramdisk is set to non-existing file '%s' on %s", sysinit.Args[ramdiskOption], bootPartition)
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

		bootMsg = "kernel exec"
	} else {
		bootMsg = "kernel load"
	}

	bootMsg = fmt.Sprintf("%s failed with status %d\n", bootMsg, status.Exit)

	os.Chdir("/")
	return false
}

func wait() bool {
	if !sysinit.AsInit() {
		return true
	}

	ch := make(chan bool)
	go func() {
		time.Sleep(time.Second)
		ch <- true
	}()
	go func() {
		fmt.Scanln()
		ch <- false
	}()
	return <-ch
}

func main() {
	resetBootOptions()
	sysinit.Init()

	if sysinit.AsInit() {
		fmt.Print(`                                       		

      _/_/_/  _/        _/            _/                              _/      
   _/              _/_/_/    _/_/    _/_/_/      _/_/      _/_/    _/_/_/_/   
    _/_/    _/  _/    _/  _/_/_/_/  _/    _/  _/    _/  _/    _/    _/        
       _/  _/  _/    _/  _/        _/    _/  _/    _/  _/    _/    _/         
_/_/_/    _/    _/_/_/    _/_/_/  _/_/_/      _/_/      _/_/        _/_/      
`)
	}

	defer sysinit.Exit()
	if tryBoot() {
		return
	}

	fmt.Printf("%s...\n", bootMsg)
	sysinit.DebugShell()
}
