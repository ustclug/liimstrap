#!/bin/sh

UPTIME="$(cut -d. -f1 /proc/uptime)"
NETDEV="$(ip -j route get 202.38.93.94 | jq -r .[].dev)"
test -n "$NETDEV" && MACADDR="$(tr -d : < "/sys/class/net/$NETDEV/address")"
VERSION="$(grep -Po "version=\K\S*" /proc/cmdline)"
test -z "$VERSION" && VERSION="devel"

exec curl -s http://pxe.ustc.edu.cn:3000/ -X POST \
  -d "mac=$MACADDR&version=$VERSION&uptime=$UPTIME"
