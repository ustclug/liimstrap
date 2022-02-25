# liimstrap

中国科大图书馆图书查询机自动生成脚本，当前版本基于 Debian Bullseye 开发。

话说，LIIMS 是嘛意思？我猜是 Library Independent Inquery Machine System 吧。

## 依赖

使用相同版本的 Debian，安装以下软件包：

```shell
$ sudo apt install debootstrap build-essential libcurl4-openssl-dev libx11-dev libxext-dev libxss-dev curl
```

## 生成

```sh
sudo ./liimstrap [ROOT]
```

[ROOT]是一个存放镜像根文件系统的目录。

可以把 root 密码放到一个名为 .rootpasswd 的文件里，该文件和 liimstrap 脚本放在同一级文件夹下。

`etc/authorized_keys` 文件里放的是 root 远程 SSH 登录的公钥。

## 压成 SqaushFS 镜像

```sh
sudo ./deploy [ROOT] [DEST]
```

会在 `[DEST]` 中创建一个名为 `liims<日期>` 的目录，下有三个文件：

- `vmlinuz` 是内核
- `initrd.img` 是 initrd
- `root.sfs` 是根目录的镜像

Grub 配置参见 `grub.example` 文件。

## 从 Docker 构建

```sh
# docker build -t ustclug/liimstrap:liims-2 .
# docker run -it --privileged --cap-add=SYS_ADMIN --rm -v $DATA_PATH:/srv/dest -e ROOT_PASSWORD=test ustclug/liimstrap:liims-2  # 此命令创建 rootfs 文件
# docker run -it --privileged --cap-add=SYS_ADMIN --rm -v $DATA_PATH:/srv/dest -e ROOT_PASSWORD=test -e SQUASHFS=true ustclug/liimstrap:liims-2  # 此命令创建 rootfs 文件并打包为 squashfs
```

## 本地调试

1. 安装 NFS Server（`nfs-kernel-server`）
2. 配置 `/etc/exports` 如下：

   ```
   /liims	localhost(ro,no_root_squash,async,insecure,no_subtree_check)
   ```

   其中 /liims 是生成文件所在的目录。编辑完成后，执行 `exportfs -ra` 命令更新。

3. 使用以下参数启动 qemu：

   ```
   qemu-system-x86_64 -kernel ./vmlinuz -initrd ./initrd.img -m 700m -machine accel=kvm -append "nfsroot=10.0.2.2:/liims ip=dhcp boot=nfs"
   ```

   你的当前所在目录需要有生成的 vmlinuz 和 initrd.img 文件，内存等参数可以按需调整。

   如果是 squashfs，那么参数对应是：

   ```
   qemu-system-x86_64 -kernel ./vmlinuz -initrd ./initrd.img -m 700m -machine accel=kvm -append "nfsroot=10.0.2.2:/liims ip=dhcp boot=nfs squashfs=root.sfs"
   ```
