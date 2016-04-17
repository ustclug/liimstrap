# liimstrap

中国科大图书馆图书查询机自动生成脚本，基于 Arch Linux 开发。

话说，LIIMS 是嘛意思？我猜是 Library Independent Inquery Machine System 吧。

## 依赖

首先，必须要用 Arch Linux。然后需要安装以下软件包：

```
pacman -S arch-install-scripts base-devel squashfs-tools
```

## 生成

```
sudo ./liimstrap [ROOT]
```

可以把 root 密码放到一个名为 .rootpasswd 的文件里，该文件和 liimstrap 脚本放在同一级文件夹下。

`etc/authorized_keys` 文件里放的是 root 远程 SSH 登录的公钥。

## 压成 SqaushFS 镜像

```
sudo ./deploy [ROOT] [DEST]
```

会在 `[DEST]` 中创建一个名为 `liims<日期>` 的目录，下有三个文件：
* `vmlinuz` 是内核
* `initrd.img` 是 initrd
* `root.sfs` 是根目录的镜像

PXELINUX 配置参见 `pxelinux.cfg.example` 文件。
