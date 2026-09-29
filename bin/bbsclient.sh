#!/bin/bash
set -e
# This script is the main process of liims-bbs.service. On any exit, systemd
# kills the entire control group (including luit/telnet and swayidle).
swayidle -w timeout 60 'systemctl --user --no-block stop liims-bbs.service' &
foot --fullscreen --app-id=liims-bbs -e luit -encoding GBK telnet bbs.ustc.edu.cn &
# A broken idle monitor must not leave an unmonitored BBS session behind.
wait -n
