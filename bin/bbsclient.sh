#!/bin/bash

X_DISPLAY=":0"
X_AUTHFILE="/home/$LIIMSUSER/.Xauthority"

echo "Start BBS client now..."
xterm -en gbk -fullscreen -cjk_width -rv \
      -fa "Terminus:style=Regular:size=14" \
      -fd "WenQuanYi Bitmap Song:size=14" \
      -e telnet bbs.ustc.edu.cn &
PID=$!

while true; do
    sleep 5

    IDLE=$(xidle $X_DISPLAY $X_AUTHFILE || echo 0)
    echo "> X idle time: $IDLE"
    
    if ps -p $PID > /dev/null; then
        echo "> subprocess still running..."
        test $IDLE -ge 60000 && break
    else
        echo "Closed by user!"
        exit 0
    fi
done

echo "Automatic close after timeout..."
kill $PID
echo "Done!"
