#!/bin/bash

# 1/4 的概率重启
if [[ $(($(date +%s)%4)) < 2 ]]; then
    echo "Not reboot today"
else
    # 重启之前随机等待若干秒重启
    SLEEPTIME=$((RANDOM%240))
    echo "Will reboot after $SLEEPTIME seconds."
    sleep $SLEEPTIME
    reboot
fi
