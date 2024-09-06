#!/bin/sh
set -eux
cd $(dirname $0)

loglevel=1
boot_param=""
board_arg=""
output="boot/bootloader.bin"
case "${1:-}" in 
	"")
		loglevel=7
		board_arg="--board arm64-generic"
		output="boot/bootloader.debug.bin"
	;;

	*)
		boot_param="boot=$1"
	;;
esac

boot_param="console=tty1 loglevel=$loglevel $boot_param"

depthchargectl build $board_arg \
	--output $output \
	--root none \
	--boot-mountpoint none \
	--root-mountpoint none \
	--kernel-cmdline "$boot_param" \
	--kernel kernel/vmlinuz \
	--fdtdir kernel/dtbs \
	--initramfs boot/initramfs.cpio.xz
