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

squashfs=
for x in $(cat /proc/cmdline); do
  case "$x" in
    squashfs=*) eval "$x";;
    *);;
  esac
done

mkdir /ro /rw /overlay
if test -n "$squashfs"; then
  mount -t squashfs -o ro "$rootmnt/$squashfs" /ro
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
