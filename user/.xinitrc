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
systemctl --user start pulseaudio
sleep 2
/usr/bin/pacmd set-sink-port `pacmd list-sinks | sed -e '/name:/!d' | head -n 1 | sed -e 's/.*<\(.*\)>/\1/'` analog-output-headphones
exec openbox-session
