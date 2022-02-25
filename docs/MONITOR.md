# Monitor 简介

See <https://docs.ustclug.org/services/pxe/liims/#monitor>.

之前的 monitor 是一个装在 Docker 容器里的小 Python 脚本，现在是一个 Go 程序（使用 Systemd 保障安全性）。需要注意它是无状态的，所以服务一重启机器状态信息就清空了。
