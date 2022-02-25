# LIIMS image 内容介绍

本 LIIMS 版本基于 Debian，下面将对其构建内容做一个简单的技术介绍。

## 查询机启动时发生了什么？

查询机启动时，首先 PXE 启动，PXE 服务器会为查询机提供 GRUB 程序。GRUB 配置中包含根据 MAC 地址分配不同设置的配置，如果查询机的 MAC 地址符合，则加载对应的配置项，直接启动 Linux。配置项文件内容大致与 grub.example 相似。

GRUB 会下载 initrd 和 vmlinuz，然后将控制权转交给它们。Initrd 会根据启动参数进行必须的配置（加载 rootfs），之后正常启动系统。

启动参数中的 `boot=nfs` 会被 Debian 的 initramfs-tools 脚本解析为需要根据 `nfsroot` 和 `ip` 参数配置网络并挂载 NFS 对应路径到 `$rootmnt`，之后 `scripts/init-bottom/overlay.sh` 会执行，在已有的基础上挂载 squashfs（如果需要），并且加上 overlay，使得最终的 rootfs 可以读写。否则由于 NFS 是只读的，rootfs 会变成 read-only 的，会导致系统运行出现错误。

此外，启动参数可以被程序从 `/proc/cmdline` 读取，自定义程序也会使用。

## 查询机辅助脚本工具（`bin`）

`bin` 目录下放置了一些小程序：

- `xidle.c`：根据 X 的屏保接口判断系统闲置时间并输出（需要 xauth）
- `reset.sh`：重置系统 /home 目录内容，重启图形界面；目前在 `etc/root.crontab` 里面配置了每半小时执行一次，执行时循环使用 xidle 判断闲置时间，如果大于 30000 毫秒，则启动重置逻辑
- `heartbeat.sh`：向 pxe.ustc.edu.cn:3000 发送心跳包，由 systemd user timer 执行
- `chameleon.sh`：如果 `/proc/cmdline` 设置了 profile，则根据 profile 的值修改 midori 配置，用于先研院查询机（需要不同的配置）
- `bbsclient.sh`：基于 xterm 的简单 BBS 客户端，闲置 60000 毫秒后会关闭

## 网络访问限制

网络访问限制通过 hosts 和 iptables 实现，仅限制 liims 用户。

## SSH

TBD

## netdata

TBD

## Midori 配置

Midori 上游已经不再维护，因为有一些 feature（例如自动重置）不太容易用别的浏览器实现，目前使用魔改版的 midori：<https://github.com/taoky/midori>。Vala 语法类似于 C#，改起来不算难。

主要的配置有三处：

- `~/.config/midori/config`: 主配置，包括主页、搜索方式、启用扩展等
- `~/.local/share/midori/extensions`: 扩展，可以在页面上加 CSS/JS；格式和其他的不那么兼容，需要手写。目前启用了两个扩展：
    - liims：提醒用户可以用 Ctrl + Space 切换输入法（midori 的 `alert()` UI 很优雅，不是直接弹框，而是只在 URL 框下面提示）
    - cssfix：修复 Debian 11 中 libwebkit2gtk 中文伪粗体很丑的 bug
- `~/.config/openbox/rc.xml`: OpenBox 窗口管理器配置，包含了让 Midori 没有窗口边框且最大化的配置

理论上后端引擎 WebKit 不是很老，所以大部分普通的网页应该不会有特别离谱的兼容性问题。

## fcitx 输入法

Panel 上的按钮实际上调用的是 `fcitx-remote` 程序（可以在 `~/.config/fbpanel/default` 看到），大部分时候应该是能用的，虽然 Linux 上配置输入法确实有点玄学。

搜狗输入法是商业软件，软件源里没有，所以需要自己配置，所幸目前的版本的 deb 安装之后再补上缺的两个依赖 `libasound2 libgomp1` 就能用，不再像之前的版本需要自己把 `sogou-qimpanel` 开出来。

## Systemd user service

在通过终端（Ctrl + Alt + T）进入 root shell 之后，如果需要调试 systemd user service，直接 `su liims` 即可。

所有的 user service 都是在 `~/.xinitrc` 里面启动的。此外 enabled timer 需要软链接，否则新版本的 systemd 不认账。
