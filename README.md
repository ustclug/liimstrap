# liimstrap

中国科大图书馆图书查询机自动生成脚本，当前版本基于 Debian 13（Trixie）amd64，使用 labwc/Wayland、Waybar、Fcitx 5 和 LIIMS Browser。

话说，LIIMS 是嘛意思？我猜是 Library Independent Inquery Machine System 吧。

## 依赖

参见 [Dockerfile](Dockerfile)。

## 生成

```sh
sudo ./liimstrap <ROOT>
```

ROOT 是存放镜像根文件系统的目录。

镜像的 root 密码可以通过 `ROOT_PASSWORD` 环境变量提供。

SSH 使用 `etc/ssh/ssh_user_ca` 配置的证书 CA。

## 压成 SquashFS 镜像

```sh
sudo ./deploy <ROOT> <DEST>
```

会在 DEST 目录中创建四个文件：

- `vmlinuz` 是内核
- `initrd.img` 是 initrd
- `root.sfs` 是根目录的镜像
- `SHA256SUMS` 是上述三个文件的 SHA-256 校验值

GRUB 配置参见 `grub.example` 文件。

## 从 Docker 构建

```sh
docker build -t ustclug/liimstrap:liims-3 .
# 此命令创建 rootfs 内容
docker run -it --privileged --rm -v $DATA_PATH:/srv/dest -e ROOT_PASSWORD=test ustclug/liimstrap:liims-3
# 此命令创建 rootfs 内容并打包为 squashfs
docker run -it --privileged --rm -v $DATA_PATH:/srv/dest -e ROOT_PASSWORD=test -e SQUASHFS=true ustclug/liimstrap:liims-3
```

## 本地调试

Rootless Podman 生成的 rootfs 在宿主机上使用映射后的 UID/GID，不能直接通过
NFS 作为根目录启动，也不能在宿主机上直接用 `sudo ./deploy` 打包。
需要在同一用户的 Podman 用户命名空间中打包，恢复镜像里的 UID/GID：

```sh
mkdir -p /path/to/artifacts
podman unshare ./deploy /path/to/rootfs /path/to/artifacts
```

也可以在构建容器中设置 `SQUASHFS=true`，直接输出镜像。
然后导出 artifacts 目录，使用下面带 `squashfs=root.sfs` 的 QEMU 命令。
直接 NFS 启动 rootfs 仅适用于磁盘上的 UID/GID 已与目标系统一致的构建产物。

### HTTP 启动

在包含 `vmlinuz`、`initrd.img` 和 `root.sfs` 的目录启动 HTTP 服务：

```sh
python3 -m http.server 8000
```

另一个终端中启动 QEMU：

```sh
qemu-system-x86_64 -kernel ./vmlinuz -initrd ./initrd.img -m 2g -device virtio-vga -machine accel=kvm -append "ip=dhcp boot=http root_sfs=http://10.0.2.2:8000/root.sfs"
```

HTTP 启动会将 `root.sfs` 下载到内存；根据镜像大小调整 `-m`。

### NFS 启动

安装 `nfs-kernel-server`，将镜像目录通过 `/etc/exports` 导出，例如：

```sh
/liims localhost(ro,no_root_squash,async,insecure,no_subtree_check)
```

执行 `exportfs -ra` 后，用 NFS 上的 SquashFS 启动：

```sh
qemu-system-x86_64 -kernel ./vmlinuz -initrd ./initrd.img -m 2g -device virtio-vga -machine accel=kvm -append "nfsroot=10.0.2.2:/liims ip=dhcp boot=nfs squashfs=root.sfs"
```

若 NFS 根目录本身已保存正确 UID/GID，也可以去掉 `squashfs=root.sfs` 直接启动。

## 技术细节

见 `docs` 目录。
