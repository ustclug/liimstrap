#!/bin/bash

pacman -Syu --noconfirm

cd /opt/liims
echo $ROOT_PASSWORD > .rootpasswd
mkdir -p /srv/root
mkdir -p /srv/dest
./liimstrap /srv/root
./deploy /srv/root /srv/dest

