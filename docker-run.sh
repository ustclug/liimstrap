#!/bin/bash -e

mkdir -p /srv/dest

if [ -n "$SQUASHFS" ]; then
    echo "Build to squashfs."
    mkdir -p /srv/root
    ./liimstrap /srv/root
    ./deploy /srv/root /srv/dest
else
    ./liimstrap /srv/dest
fi
