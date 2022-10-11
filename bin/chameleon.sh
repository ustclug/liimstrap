#!/bin/sh

profile=""
for x in $(cat /proc/cmdline); do
  case "$x" in
    profile=*) eval "$x";;
    *) ;;
  esac
done

case "$profile" in
  "") ;;
  iat)
    F="$HOME/.config/midori/config"
    sed -i 's|\(^homepage=\).*$|homepage=http://pxe.ustc.edu.cn/liims/index_iat.html|g' "$F"
    sed -i 's|http://opac\.lib\.ustc\.edu\.cn|http://iat.lib.ustc.edu.cn|g' "$F"
    ;;
  *) printf "Unknown profile %s\n" "$profile" >&2;;
esac
