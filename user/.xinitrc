#!/bin/sh

xset s off
xset -dpms
xset s noblank

[ -f /etc/xprofile ] && source /etc/xprofile
[ -f ~/.xprofile ] && source ~/.xprofile

fbpanel &
xcompmgr -n &
systemctl --user start chameleon.service
systemctl --user start midori.service
systemctl --user start fcitx.service
exec openbox-session
