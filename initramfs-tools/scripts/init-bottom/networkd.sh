#!/bin/sh

case "$1" in
  prereqs) exit 0 ;;
esac

# /run survives the initramfs handoff. Configure only the interface used to
# fetch the root filesystem, including boots with a static ip= setting.
boot_conf=
boot_device=$(awk '$2 == "00000000" && $8 == "00000000" { print $1; exit }' /proc/net/route)
if [ -n "$boot_device" ] && [ -f "/run/net-$boot_device.conf" ]; then
  boot_conf="/run/net-$boot_device.conf"
else
  for conf in /run/net-*.conf; do
    if [ -f "$conf" ]; then
      boot_conf=$conf
      break
    fi
  done
fi
[ -n "$boot_conf" ] || exit 0

# These files are produced and sourced by initramfs-tools itself.
DEVICE= PROTO= IPV4ADDR= IPV4NETMASK= IPV4GATEWAY= IPV4DNS0= IPV4DNS1= DNSDOMAIN= DOMAINSEARCH=
. "$boot_conf"
case "$DEVICE" in
  ""|*[!a-zA-Z0-9_.:-]*) echo "Invalid boot network interface: $DEVICE" >&2; exit 1 ;;
esac

network_dir=/run/systemd/network
mkdir -p "$network_dir"
network_file="$network_dir/05-liims-boot.network"
{
  printf '[Match]\nName=%s\n\n[Network]\nIPv6AcceptRA=yes\n' "$DEVICE"
  case "$PROTO" in
    dhcp|bootp)
      printf 'DHCP=ipv4\n'
      ;;
    *)
      if [ -n "$IPV4ADDR" ] && [ -n "$IPV4NETMASK" ]; then
        prefix=0
        rest=$IPV4NETMASK
        while [ -n "$rest" ]; do
          octet=${rest%%.*}
          if [ "$rest" = "$octet" ]; then rest=; else rest=${rest#*.}; fi
          case "$octet" in
            255) bits=8 ;; 254) bits=7 ;; 252) bits=6 ;; 248) bits=5 ;;
            240) bits=4 ;; 224) bits=3 ;; 192) bits=2 ;; 128) bits=1 ;;
            0) bits=0 ;; *) echo "Invalid boot netmask: $IPV4NETMASK" >&2; exit 1 ;;
          esac
          prefix=$((prefix + bits))
        done
        printf 'Address=%s/%s\n' "$IPV4ADDR" "$prefix"
      fi
      [ -z "$IPV4GATEWAY" ] || printf 'Gateway=%s\n' "$IPV4GATEWAY"
      ;;
  esac
  # for dns in $IPV4DNS0 $IPV4DNS1; do
  #   [ "$dns" = 0.0.0.0 ] || printf 'DNS=%s\n' "$dns"
  # done
  # for domain in $DOMAINSEARCH; do
  #   printf 'Domains=%s\n' "$domain"
  # done
  if [ -z "$DOMAINSEARCH" ] && [ -n "$DNSDOMAIN" ]; then
    printf 'Domains=%s\n' "$DNSDOMAIN"
  fi
  if [ "$PROTO" = dhcp ] || [ "$PROTO" = bootp ]; then
    printf '\n[DHCPv4]\nClientIdentifier=mac\n'
  fi
} > "$network_file"
