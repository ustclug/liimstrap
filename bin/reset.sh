#!/bin/bash

X_DISPLAY=":0"
X_AUTHFILE="/home/$LIIMSUSER/.Xauthority"

: "${LIIMSUSER:=liims}" "${IDLETIMEOUT:=30000}"

while true; do
  IDLE=$(/liims/bin/xidle "$X_DISPLAY" "$X_AUTHFILE" || echo 32000)
  echo "> X idle time: $IDLE"
  test "$IDLE" -ge "$IDLETIMEOUT" && break
  sleep 5
done

echo "Now restart user slim..."
XDG_RUNTIME_DIR="/run/user/$(id -u "$LIIMSUSER")" \
  su "$LIIMSUSER" -c 'systemctl --user exit'
systemctl stop slim.service
sleep 2

echo "Reset /home..."
su "$LIIMSUSER" -c "rsync -a --delete /ro/home/$LIIMSUSER/ /home/$LIIMSUSER/"
sleep 2

systemctl daemon-reload
systemctl start slim.service
echo "Done!"
