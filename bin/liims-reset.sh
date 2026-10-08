#!/bin/bash
set -euo pipefail

: "${LIIMSUSER:=liims}"

fail() {
  echo "LIIMS reset: $*" >&2
  return 1
}

# Exactly one local greetd session; SSH/TTY sessions never supply idle state.
find_session() {
  local sessions candidate properties name service remote seat found=
  sessions=$(loginctl list-sessions --no-legend --no-pager) || return 1
  while read -r candidate _; do
    [[ -n "$candidate" ]] || continue
    properties=$(loginctl show-session "$candidate" -p Name -p Service -p Remote -p Seat) || return 1
    name='' service='' remote='' seat=''
    while IFS='=' read -r key value; do
      case "$key" in
        Name) name=$value ;;
        Service) service=$value ;;
        Remote) remote=$value ;;
        Seat) seat=$value ;;
      esac
    done <<< "$properties"
    # greetd uses greetd-greeter for default_session (our desktop autologin).
    if [[ "$name" = "$LIIMSUSER" && "$remote" = no && "$seat" = seat0 &&
          ( "$service" = greetd || "$service" = greetd-greeter ) ]]; then
      [[ -z "$found" ]] || return 1
      found=$candidate
    fi
  done <<< "$sessions"
  [[ -n "$found" ]] || return 1
  printf '%s\n' "$found"
}

check_monitor() {
  local session=$1 pid scope command
  read -r pid < "$runtime/liims-idle.pid" || return 1
  [[ "$pid" =~ ^[0-9]+$ ]] || return 1
  [[ $(stat -c %u "/proc/$pid") = "$uid" ]] || return 1
  command=$(readlink "/proc/$pid/exe") || return 1
  [[ "$command" = /usr/bin/swayidle ]] || return 1
  scope=$(loginctl show-session "$session" -p Scope --value) || return 1
  [[ -n "$scope" ]] || return 1
  # A stale PID or a monitor from another session must not authorize a reset.
  grep -Fq "/$scope" "/proc/$pid/cgroup" || return 1
}

wait_for_idle() {
  local session initial_session idle
  initial_session=$(find_session) || { fail 'no unique local greetd session; skipping'; return 1; }
  while true; do
    session=$(find_session) || { fail 'login session disappeared; skipping'; return 1; }
    [[ "$session" = "$initial_session" ]] || { fail 'login session changed; skipping'; return 1; }
    check_monitor "$session" || { fail 'idle monitor unavailable; skipping'; return 1; }
    idle=$(loginctl show-session "$session" -p IdleHint --value) || return 1
    case "$idle" in
      yes) break ;;
      no) sleep 5 ;;
      *) fail 'invalid idle state; skipping'; return 1 ;;
    esac
  done

  printf '%s\n' "$session"
}

main() {
  [[ "$EUID" = 0 ]] || { fail 'must run as root'; return 1; }
  exec 9>/run/lock/liims-reset.lock
  flock -n 9 || return 0

  uid=$(id -u "$LIIMSUSER")
  runtime="/run/user/$uid"
  local session
  session=$(wait_for_idle) || return 1

  [[ -d "/ro/home/$LIIMSUSER" && ! -L "/home/$LIIMSUSER" ]] || {
    fail 'missing baseline or unsafe home path'; return 1;
  }
  # Recheck immediately before stopping the session.
  check_monitor "$session" || return 1
  [[ $(loginctl show-session "$session" -p IdleHint --value) = yes ]] || return 0
  echo 'Stopping graphical login and all LIIMS user processes...'
  systemctl stop greetd.service
  # greetd may already have closed the PAM session; stopping user@ explicitly
  # also handles logind's delayed user-manager shutdown.
  if loginctl show-user "$uid" >/dev/null 2>&1; then
    loginctl terminate-user "$LIIMSUSER"
  fi
  systemctl stop "user@$uid.service"
  for ((attempt=0; attempt<50; attempt++)); do
    if ! pgrep -u "$uid" >/dev/null; then break; fi
    sleep 0.1
  done
  if pgrep -u "$uid" >/dev/null; then
    fail 'user processes remain; leaving greetd stopped'; return 1
  fi

  echo 'Restoring home from the read-only image...'
  rsync -a --delete -- "/ro/home/$LIIMSUSER/" "/home/$LIIMSUSER/"
  systemctl start greetd.service
  echo 'LIIMS reset completed.'
}

if [[ "${BASH_SOURCE[0]}" = "$0" ]]; then
  main "$@"
fi
