#!/bin/sh
set -eux
cd $(dirname $0)
find init lib libexec -print0 | cpio -o -d -H newc > boot/initramfs.cpio
xz -kf -9 --check=crc32 boot/initramfs.cpio
