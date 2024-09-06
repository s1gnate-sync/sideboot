sideboot
--------

Universal bootloader for you ARM64 chromebook. 

Run `./mkinitfs.sh` to regenerate

Run `./mkbootloader.sh <BOOT_PART>` where `BOOT_PART` is your uefi (/boot) partition e.g `./mkbootloader.sh /dev/mmcblk1p2`,
create `/boot/sideboot.cfg` file on your boot partition and provide boot arguments:
```
-I <INITRAMFS_NAME> -K <KERNEL_NAME> -- <KERNEL_CMDLINE>
```

for example:
```
-Iinitramfs -- console=tty1 quiet loglevel=1 root=UUID=f402667c-44e3-448d-b3d4-f8e8796bae9d PMOS_NOSPLASH
```

for debug run `./mkbootloader.sh` without arguments, it will create generic kpart with shell as default action.

deps
----

- `busybox` https://pkgs.alpinelinux.org/package/edge/main/aarch64/busybox-static
- `kexec` https://pkgs.alpinelinux.org/package/edge/community/aarch64/kexec-tools
- `kernel` https://github.com/FyraLabs/submarine (https://nightly.link/FyraLabs/submarine/workflows/build/main/submarine-arm64.zip)
