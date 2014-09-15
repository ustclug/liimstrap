#!/bin/sh

xset s off
xset -dpms
xset s noblank

[ -f /etc/xprofile ] && source /etc/xprofile
[ -f ~/.xprofile ] && source ~/.xprofile

fbpanel &
xcompmgr -n &
systemctl --user start midori
systemctl --user start fcitx
systemctl --user start sogou-qimpanel
exec openbox-session
