#!/bin/sh

PREREQ=""
prereqs()
{
	printf "%s\n" "$PREREQ"
}
case $1 in
# get pre-requisites
prereqs)
	prereqs
	exit 0
	;;
esac

# default threshold 3.3 GiB
squashfs=
squashfs_minmem=3460300
for x in $(cat /proc/cmdline); do
  case "$x" in
    squashfs=*) eval "$x";;
    squashfs_minmem=*) eval "$x";;
    *);;
  esac
done

mkdir /ro /rw /overlay
if test -n "$squashfs"; then
  sfsfile="$rootmnt/$squashfs"
  sysmem="$(awk '/MemTotal/{print $2}' /proc/meminfo)"
  if test "$sysmem" -ge "$squashfs_minmem"; then
    printf "enough system RAM, copy squashfs to tmpfs\n"
    size="$(stat -c %s "$sfsfile")"
    mkdir /tmpfs
    mount -t tmpfs -o size="$((size + 20480000))" tmpfs /tmpfs
    cp "$sfsfile" /tmpfs/root.sfs
    mount -t squashfs -o ro /tmpfs/root.sfs /ro
    umount -l /tmpfs
  else
    # mount directly from NFS
    mount -t squashfs -o ro "$sfsfile" /ro
  fi
else
  mount --move "$rootmnt" /ro
fi
mount -t tmpfs tmpfs /rw -o noatime
mkdir /rw/upper /rw/work
mount -t overlay -o noatime,lowerdir=ro,upperdir=rw/upper,workdir=rw/work overlay /overlay
mkdir -p /overlay/rw /overlay/ro
mount --move /ro /overlay/ro
mount --move /rw /overlay/rw
mount --move /overlay "$rootmnt"
