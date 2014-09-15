#!/bin/bash

X_DISPLAY=":0"
X_AUTHFILE="/home/$LIIMSUSER/.Xauthority"

echo "Start BBS client now..."
env LC_ALL=zh_CN.GBK \
    sakura -m -x \
    "bash -c 'echo Please wait 10 seconds...; telnet -4 bbs.ustc.edu.cn'" &
PID=$!

while true; do
    sleep 5

    IDLE=$(xidle $X_DISPLAY $X_AUTHFILE || echo 0)
    echo "> X idle time: $IDLE"
    
    if ps -p $PID > /dev/null; then
        echo "> subprocess still running..."
        test $IDLE -ge 60000 && break
    else
        "Closed by user!"
        exit 0
    fi
done

echo "Automatic close after timeout..."
kill $PID
echo "Done!"

