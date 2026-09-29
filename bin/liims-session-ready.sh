#!/bin/bash
set -e
: "${WAYLAND_DISPLAY:?labwc must export WAYLAND_DISPLAY}"
: "${XDG_SESSION_ID:?Missing login session}"
unset DISPLAY
systemctl --user unset-environment DISPLAY
dbus-update-activation-environment --systemd WAYLAND_DISPLAY XDG_SESSION_TYPE \
  XDG_CURRENT_DESKTOP LANG LC_ALL GTK_IM_MODULE
systemctl --user import-environment WAYLAND_DISPLAY XDG_SESSION_TYPE \
  XDG_CURRENT_DESKTOP XDG_SESSION_ID LANG LC_ALL GTK_IM_MODULE
systemctl --user reset-failed
systemctl --user start liims-session.target

# Keep swayidle inside the PAM session so logind resolves the right session.
# The reset service checks this PID and its session scope before trusting IdleHint.
printf '%s\n' "$$" > "$XDG_RUNTIME_DIR/liims-idle.pid"
exec swayidle -w idlehint 90
