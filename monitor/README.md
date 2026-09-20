# liims-monitor

See <https://docs.ustclug.org/services/pxe/liims/#monitor>.

之前的 monitor 是一个装在 Docker 容器里的小 Python 脚本，现在是一个 Go 程序（使用 Systemd 保障安全性）。机器状态保存在 `/var/lib/liims-monitor/state.db` SQLite 数据库中。
