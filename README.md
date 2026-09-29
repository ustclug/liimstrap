# liimstrap

中国科大图书馆图书查询机自动生成脚本，当前版本基于 Debian Bookworm 开发。

话说，LIIMS 是嘛意思？我猜是 Library Independent Inquery Machine System 吧。

## 依赖

参见 [Dockerfile](Dockerfile)。

## 生成

```sh
sudo ./liimstrap <ROOT>
```

ROOT 是存放镜像根文件系统的目录。

镜像的 root 密码可以通过 `ROOT_PASSWORD` 环境变量提供。

`etc/authorized_keys` 文件里放的是 root 远程 SSH 登录的公钥。

## 压成 SqaushFS 镜像

```sh
sudo ./deploy <ROOT> <DEST>
```

会在 DEST 目录中创建三个文件：

- `vmlinuz` 是内核
- `initrd.img` 是 initrd
- `root.sfs` 是根目录的镜像

GRUB 配置参见 `grub.example` 文件。

## 从 Docker 构建

```sh
# docker build -t ustclug/liimstrap:liims-2 .
# docker run -it --privileged --rm -v $DATA_PATH:/srv/dest -e ROOT_PASSWORD=test ustclug/liimstrap:liims-2  # 此命令创建 rootfs 内容
# docker run -it --privileged --rm -v $DATA_PATH:/srv/dest -e ROOT_PASSWORD=test -e SQUASHFS=true ustclug/liimstrap:liims-2  # 此命令创建 rootfs 内容并打包为 squashfs
```

## 本地调试

1. 在包含 `vmlinuz`、`initrd.img` 和 `root.sfs` 的目录启动 HTTP 服务：

   ```sh
   python3 -m http.server 8000
   ```

2. 在另一个终端中，从同一目录启动 qemu：

   ```sh
   qemu-system-x86_64 -kernel ./vmlinuz -initrd ./initrd.img -m 2g -machine accel=kvm -append "ip=dhcp boot=http root_sfs=http://10.0.2.2:8000/root.sfs"
   ```

   `root.sfs` 会下载到内存，内存容量需要同时容纳压缩镜像和运行中的系统；根据实际镜像大小调整 `-m`。

### NFS 启动

安装 `nfs-kernel-server`，将镜像目录通过 `/etc/exports` 导出，例如：

```sh
/liims localhost(ro,no_root_squash,async,insecure,no_subtree_check)
```

执行 `exportfs -ra` 后，可以直接使用 NFS 根目录，或使用 NFS 上的 squashfs 镜像：

```sh
qemu-system-x86_64 -kernel ./vmlinuz -initrd ./initrd.img -m 700m -machine accel=kvm -append "nfsroot=10.0.2.2:/liims ip=dhcp boot=nfs"
qemu-system-x86_64 -kernel ./vmlinuz -initrd ./initrd.img -m 700m -machine accel=kvm -append "nfsroot=10.0.2.2:/liims ip=dhcp boot=nfs squashfs=root.sfs"
```

## 技术细节

见 `docs` 目录。
