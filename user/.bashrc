[[ $- != *i* ]] && return
PS1='\u@\h:\w\$ '
export LC_ALL=C.UTF-8
export XDG_RUNTIME_DIR=/run/user/$(id -u)
alias ls='ls --color=auto'
alias l='ls -CF'
alias la='ls -A'
alias ll='ls -alF'
