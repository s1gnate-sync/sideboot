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

Write bootloader to your kernel partition `pv boot/bootloader.bin -o /dev/mmcblk1p1`

partition layout example
------------------------

```
Units: sectors of 1 * 512 = 512 bytes
Sector size (logical/physical): 512 bytes / 512 bytes
I/O size (minimum/optimal): 512 bytes / 512 bytes
Disklabel type: gpt
Disk identifier: C1330B1D-CFD4-4A41-AAC0-33D9180978E4
First usable LBA: 34
Last usable LBA: 122142686
Alternative LBA: 122142719
Partition entries starting LBA: 2
Allocated partition entries: 128
Partition entries ending LBA: 33

/dev/mmcblk1p1    8192    270335    262144 FE3A2A5D-4F32-41A7-B725-ACCC3285A309 6F7293D2-17F8-EF4E-953A-C0DE3D06670D DEPTHCHARGE_KERNEL GUID:49,51,52,54,56
/dev/mmcblk1p2  270336   1318911   1048576 C12A7328-F81F-11D2-BA4B-00A0C93EC93B C6425A07-B794-F14D-954F-FD8FB6A7F97F DISTRO_BOOT   
/dev/mmcblk1p3 1318912 122142686 120823775 EBD0A0A2-B9E5-4433-87C0-68B6B72699C7 AC49AE92-FC2E-4A00-A462-6A1C9F36D10F DISTRO_ROOT
```

deps
----

depthcharge-tools (https://github.com/alpernebbi/depthcharge-tools/tree/master) is required for kpart building

- `busybox` https://pkgs.alpinelinux.org/package/edge/main/aarch64/busybox-static
- `kexec` https://pkgs.alpinelinux.org/package/edge/community/aarch64/kexec-tools
- `kernel` https://github.com/FyraLabs/submarine (https://nightly.link/FyraLabs/submarine/workflows/build/main/submarine-arm64.zip)
