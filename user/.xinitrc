#!/bin/sh

xset s off
xset -dpms
xset s noblank

[ -f /etc/xprofile ] && source /etc/xprofile
[ -f ~/.xprofile ] && source ~/.xprofile

fbpanel &
xcompmgr -n &
systemctl --user start firefox
systemctl --user start fcitx
systemctl --user start sogou-qimpanel
systemctl --user start pulseaudio
sleep 2
/usr/bin/pacmd set-sink-port `pacmd list-sinks | sed -ne '/name:/p' | sed -ne '1p' | sed -e 's/.*<\(.*\)>/\1/'` analog-output
pacmd list-sinks | sed -ne '/name:/p' | sed -e 's/.*<\(.*\)>/\1/' | while read sink; do pacmd set-sink-volume $sink 0x10000; done
/usr/bin/pacmd set-source-port `pacmd list-sources | sed -ne '/name:/p' | sed -ne '2p' | sed -e 's/.*<\(.*\)>/\1/'` analog-input-mic
pacmd list-sources | sed -ne '/name:/p' | sed -e 's/.*<\(.*\)>/\1/' | while read source; do pacmd set-source-volume $source 0x5000; done
exec openbox-session
