#!/bin/bash
set -e

: "${XDG_RUNTIME_DIR:?PAM must create a user runtime directory}"
: "${XDG_SESSION_ID:?PAM must create a logind session}"
export DBUS_SESSION_BUS_ADDRESS="unix:path=$XDG_RUNTIME_DIR/bus"
export XDG_SESSION_TYPE=wayland XDG_CURRENT_DESKTOP=labwc
export LANG=zh_CN.UTF-8 LC_ALL=zh_CN.UTF-8
export GTK_IM_MODULE=fcitx
# Manage activation ourselves; never export labwc's optional X11 display.
export LABWC_UPDATE_ACTIVATION_ENV=0
unset DISPLAY WAYLAND_DISPLAY

cleanup() {
  trap - EXIT
  if read -r idle_pid < "$XDG_RUNTIME_DIR/liims-idle.pid" 2>/dev/null; then
    if [[ "$idle_pid" =~ ^[0-9]+$ && $(readlink "/proc/$idle_pid/exe") = /usr/bin/swayidle ]]; then
      kill "$idle_pid" 2>/dev/null || true
    fi
  fi
  rm -f "$XDG_RUNTIME_DIR/liims-idle.pid"
  systemctl --user stop liims-session.target graphical-session.target || true
  dbus-update-activation-environment WAYLAND_DISPLAY= DISPLAY= || true
  systemctl --user unset-environment WAYLAND_DISPLAY XDG_SESSION_ID DISPLAY || true
  if [[ -n "${compositor:-}" ]]; then
    kill "$compositor" 2>/dev/null || true
    wait "$compositor" 2>/dev/null || true
  fi
}
trap cleanup EXIT
trap 'exit 0' TERM INT

# Drop any services/environment left over from a crashed compositor first.
systemctl --user stop liims-session.target graphical-session.target
systemctl --user unset-environment DISPLAY WAYLAND_DISPLAY
rm -f "$XDG_RUNTIME_DIR/liims-idle.pid"
labwc --session /usr/local/bin/liims-session-ready.sh &
compositor=$!
wait "$compositor"
