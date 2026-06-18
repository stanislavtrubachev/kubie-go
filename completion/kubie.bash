#Kubie completion script

_kubiecomplete()
{
    local cur prev

    cur=${COMP_WORDS[COMP_CWORD]}
    prev=${COMP_WORDS[COMP_CWORD-1]}

    { \unalias command; \unset -f command; } >/dev/null 2>&1 || true

    case ${COMP_CWORD} in
        1)
            cmds="ctx delete edit edit-config exec export generate-completion info lint ns update"
            COMPREPLY=($(command printf "%s\n" $cmds | command grep -e "^$cur" | command xargs))
            ;;
        2)
            case ${prev} in
                ctx|edit|delete)
                    COMPREPLY=($(command kubie ctx 2>/dev/null | command grep -e "^$cur" | command xargs))
                    ;;
                exec)
                    COMPREPLY=($(command kubie ctx 2>/dev/null | command grep -e "^$cur" | command xargs))
                    ;;
                ns)
                    COMPREPLY=($(command kubie ns 2>/dev/null | command grep -e "^$cur" | command xargs))
                    ;;
                info)
                    COMPREPLY=($(command printf "ctx\nns\ndepth\n" | command grep -e "^$cur" | command xargs))
                    ;;
                generate-completion)
                    COMPREPLY=($(command printf -- "--shell\n" | command grep -e "^$cur" | command xargs))
                    ;;
            esac
            ;;
        3)
            local prevprev=${COMP_WORDS[COMP_CWORD-2]}
            case ${prevprev} in
                exec)
                    COMPREPLY=($(command kubie exec "${prev}" default -- kubie ns 2>/dev/null | command grep -e "^$cur" | command xargs))
                    ;;
                generate-completion)
                    COMPREPLY=($(command printf "bash\nzsh\nfish\nxonsh\nnu\n" | command grep -e "^$cur" | command xargs))
                    ;;
            esac
            ;;
        *)
            COMPREPLY=()
            ;;
    esac
}

complete -F _kubiecomplete kubie
