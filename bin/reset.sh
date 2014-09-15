#!/bin/bash

X_DISPLAY=":0"
X_AUTHFILE="/home/$LIIMSUSER/.Xauthority"
SLEEPTIME=$((RANDOM%600))

echo "Sleep for $SLEEPTIME seconds..."
sleep $SLEEPTIME

while true; do
    IDLE=$(xidle $X_DISPLAY $X_AUTHFILE || echo 32000)
    echo "> X idle time: $IDLE"
    test $IDLE -ge 30000 && break
    sleep 5
done

echo "Now restart user slim..."
su $LIIMSUSER -c 'systemctl --user exit'
systemctl stop slim
sleep 2

sync
rm -rf "/aufs/rw/home"
sync

systemctl daemon-reload
systemctl start slim
echo "Done!"
