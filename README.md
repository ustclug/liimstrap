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

1. 安装 NFS Server（`nfs-kernel-server`）
2. 配置 `/etc/exports` 如下：

   ```sh
   /liims	localhost(ro,no_root_squash,async,insecure,no_subtree_check)
   ```

   其中 /liims 是生成文件所在的目录。编辑完成后，执行 `exportfs -ra` 命令更新。

3. 使用以下参数启动 qemu：

   ```sh
   qemu-system-x86_64 -kernel ./vmlinuz -initrd ./initrd.img -m 700m -machine accel=kvm -append "nfsroot=10.0.2.2:/liims ip=dhcp boot=nfs"
   ```

   你的当前所在目录需要有生成的 vmlinuz 和 initrd.img 文件，内存等参数可以按需调整。

   如果是 squashfs，那么参数对应是：

   ```sh
   qemu-system-x86_64 -kernel ./vmlinuz -initrd ./initrd.img -m 700m -machine accel=kvm -append "nfsroot=10.0.2.2:/liims ip=dhcp boot=nfs squashfs=root.sfs"
   ```

## 技术细节

见 `docs` 目录。

## 附录

### 在非科大校园网环境构建

写入 `/etc/resolv.conf` 的步骤执行后，之后的相关命令连接网络时会请求 `202.38.64.1` 作为 DNS 服务器，在校外环境下可能无法使用。这个问题可以使用 iptables 处理：

```
# 在 NAT 表中修改包的目的地（假设本地 DNS 为 127.0.0.53）
iptables -t nat -A OUTPUT -d 202.38.64.1/32 -p udp -m udp --dport 53 -j DNAT --to-destination 127.0.0.53:53
iptables -t nat -A OUTPUT -d 202.38.64.1/32 -p tcp -m tcp --dport 53 -j DNAT --to-destination 127.0.0.53:53
# 如果使用 systemd-resolved，其要求访问的 IP 必须为本地回环，但是访问 202.38.64.1 的路由不是本地回环
# 因此包的源地址也需要修改
iptables -t nat -A POSTROUTING -d 127.0.0.53/32 -p udp -m udp --dport 53 -j SNAT --to-source 127.0.0.1
iptables -t nat -A POSTROUTING -d 127.0.0.53/32 -p tcp -m tcp --dport 53 -j SNAT --to-source 127.0.0.1
```
