#!/bin/sh

PREREQ=""
prereqs()
{
	echo "$PREREQ"
}
case $1 in
# get pre-requisites
prereqs)
	prereqs
	exit 0
	;;
esac


mkdir /ro /rw /overlay
mount --move $rootmnt /ro
mount -t tmpfs tmpfs /rw -o noatime
mkdir /rw/upper /rw/work
mount -t overlay -o noatime,lowerdir=ro,upperdir=rw/upper,workdir=rw/work overlay /overlay
mkdir -p /overlay/rw /overlay/ro
mount --move /ro /overlay/ro
mount --move /rw /overlay/rw
mount --move /overlay $rootmnt
