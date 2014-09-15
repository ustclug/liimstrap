

[[ $- != *i* ]] && return
PS1='[\u@\h \W]\$ '
export XDG_RUNTIME_DIR=/run/user/$(id -u)
