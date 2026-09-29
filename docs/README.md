# LIIMS 镜像

## 启动与桌面

Debian 13 amd64，PXE/GRUB 下载 `vmlinuz` 和 `initrd.img`，通过现有
`boot=nfs`、`nfsroot`、`squashfs` 参数加载根文件系统，也支持
`boot=http root_sfs=http://.../root.sfs` 从 HTTP 下载压缩镜像到内存。
initramfs 将只读根挂到 `/ro`，内存中的可写层挂到 `/rw`，合并为 overlay 根目录。
`deploy` 输出内核、initrd、SquashFS 和 SHA256SUMS；这些文件应作为一组部署。
HTTP 启动时内存需要同时容纳压缩镜像和运行中的系统；NFS 启动可以直接使用
NFS 根目录，或挂载其中的 SquashFS，内存充足时将 SquashFS 复制到 tmpfs。

initramfs 从启动阶段的网络配置生成 `/run/systemd/network/05-liims-boot.network`。
真实系统由 `systemd-networkd` 接管网卡，`systemd-resolved` 提供 DNS，
`/etc/resolv.conf` 指向它的本地 `127.0.0.53` 解析器。

`greetd` 在 tty7 以 `liims` 用户运行 `liims-session.sh`，每次会话退出都会重新启动。
labwc 创建 Wayland socket 后，`liims-session-ready.sh` 导入实际会话环境，启动
`liims-session.target`。它关联 systemd 的 `graphical-session.target`，共同管理
浏览器、Waybar、Fcitx 5 和 heartbeat timer。退出桌面时停止整组服务。
空闲监测作为 labwc 的 session client，退出时也会结束 compositor，由 greetd 重建会话。
用户 D-Bus 和运行目录由 PAM/systemd 建立，不另开独立总线，也不启用 lingering。

不启动 Xorg，不自动锁屏或关闭显示器。Debian 的 labwc 包硬依赖 XWayland；
保留该包，但配置为按需启动且不向图形用户服务导出 DISPLAY，所有桌面应用使用原生 Wayland。目标设备必须具备可用的 DRM/KMS 驱动。
浏览器最大化并给底部面板留出空间；Win+Tab 切换窗口，Ctrl+Alt+T 打开要求 root
密码的维护终端，Print 用 grim 截图。未配置通用应用启动器或退出桌面的快捷键。

## 浏览器、面板与输入法

浏览器来自固定版本的上游 deb，安装时由 APT 解析 GTK4、libadwaita 和 WebKitGTK 6
运行依赖。镜像不包含 Rust 构建工具链。URL 与 SHA-256 必须一起更新。

`/etc/liims/browser.toml` 由浏览器包提供；内核参数 `profile=iat` 选择先研院配置，
不再运行脚本改写配置。浏览器内置首页代替旧 HTML 首页，历史 `homepage/` 文件不参与镜像构建。
配置检查：

```sh
liims-browser --check-config --profile default
liims-browser --check-config --profile iat
```

Waybar 保留 BBS、彩虹猫、浏览器重启、英语/拼音/五笔切换和时钟。
五笔使用 Debian 自带的 `wbx` 引擎。Fcitx 5 的触发键是 Ctrl+Space；GTK 使用 Fcitx 模块，foot 使用 Wayland 输入法协议。
BBS 通过 foot + luit 将 UTF-8 终端与 GBK telnet 服务相连，全桌面空闲 60 秒后关闭。
BBS 的终端、转换器、网络进程和空闲监测属于同一 systemd 控制组，退出时一起清理。
彩虹猫使用独立用户服务；重复点击不会创建多个实例。
foot 使用独立窗口，镜像禁用并屏蔽包默认启用的 foot-server service/socket。

## 网络访问限制

网络访问限制仅作用于 UID 1000（`liims`）。iptables 的 OUTPUT NAT 将其 TCP
80、3000（心跳）及 443 端口重定向到本机 GOST 的 3128 端口；filter 表只允许这些本机端口、
本机 `127.0.0.53` DNS，以及 BBS 的 Telnet 端口（固定 IP 202.38.64.3）。上游 DNS 查询由
`systemd-resolved` 发出，不受 UID 1000 的规则限制。

GOST 使用 SNI handler 检查 HTTP Host 或 HTTPS ClientHello 中的 SNI，
只有命中 `/etc/gost/bypass.txt` 的域名才会转发；它不终止 TLS，
浏览器仍直接验证目标站点的证书。`/etc/gost/hosts.txt` 包含特殊域名映射。
修改这两个文件后需重启 `liims-gost.service`。
没有 SNI 的 HTTPS 连接会被拒绝；ECH 的外层 SNI 不能证明真实目标域名，
若浏览器将来启用 ECH，需要更新策略。HTTP/3（UDP/443）会被 filter 表拒绝。

greetd 在 `netfilter-persistent`、GOST、networkd 和 resolved 启动后才启动桌面。

## 两层清理

浏览器无操作 45 秒提示、60 秒清理临时网络会话；“结束使用”也可主动清理。
清理本机浏览状态不等于注销网站服务器上的会话。

整机 timer 保持每小时 `:30` 触发、随机延迟最多 10 分钟，并等待全桌面空闲 90 秒。
`swayidle idlehint 90` 直接运行在 greetd 登录会话内。
root 重置脚本确认唯一的本地 greetd 会话、监测进程的 UID/可执行文件/会话控制组和
logind IdleHint。检测失败或会话变化时取消本轮重置；持续有人使用时最多等待 50 分钟。

重置先停止 greetd，再终止 liims 用户进程及用户管理器，确认没有残留后，使用
`rsync -a --delete` 从 `/ro/home/liims/` 恢复家目录，最后重启 greetd。
恢复失败时保持桌面停止，避免启动不完整配置；SSH 仍可维护。
重置是 root 系统服务，面板上的“重置浏览器”只重启浏览器用户服务。
每周六重启的 timer 保持不变。

## 维护与验证

heartbeat 的 HTTP 接口及周期不变，网络访问限制使用 iptables 和 GOST。
SSH 证书登录和 NTP 保留。Debian 13 官方仓库没有 netdata，本镜像不再安装它；
终端在线状态继续由 heartbeat 上报。不要在排障时关闭 WebKit sandbox。

在 liims 登录会话中可检查：

```sh
systemctl --user status liims-session.target liims-browser waybar fcitx5
journalctl --user -u liims-browser -u waybar -u fcitx5
```

root 维护入口：

```sh
journalctl -u greetd -u liims-reset
loginctl list-sessions
systemctl start liims-reset.service
```

最后一个命令会等待空闲并恢复用户目录。手动恢复失败的桌面前，应先检查错误并确认
`/home/liims` 已完整恢复，再执行 `systemctl start greetd`。

仓库检查运行 `tests/check.sh`。发布前还需完整构建，并用 QEMU virtio 显卡及代表性
真机验证启动、两个校区、中文输入、BBS、浏览会话清理、整机重置和崩溃恢复。
GUI 自动化若需独立 D-Bus，使用 `~/.local/bin/dbus-run-isolated -- <command>`；
wrapper 或 bwrap 失败时停止，不得直接运行未隔离的独立 D-Bus 会话。

新镜像先按 MAC 分配给少量终端。保留旧镜像完整目录，回滚时将 GRUB 配置切回旧目录，
不要混用两个版本的内核、initrd 和 root.sfs。
