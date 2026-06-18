set -l commands ctx delete edit edit-config exec export generate-completion info lint ns update

complete -c kubie --no-files

complete -c kubie \
    --condition "not __fish_seen_subcommand_from $commands" \
    --arguments "$commands"

set -l cmd __fish_seen_subcommand_from

complete -c kubie -n "not $cmd $commands" -l version -s V

# FIXME: This should take --kubeconfig into account
complete -c kubie -n "$cmd ctx delete edit exec; and __kubie_at_arg 1" -d 'context' \
    -a '(kubie ctx 2> /dev/null)'

complete -c kubie -n "$cmd ctx ns" -l recursive -s r -d 'spawn a new recursive shell'

complete -c kubie -n "$cmd ctx; and __kubie_at_arg 1" -a '-' -d 'switch back'
complete -c kubie -n "$cmd ctx" -l kubeconfig -s f -r -d 'load contexts from file'
complete -c kubie -n "$cmd ctx" -l namespace -s n -d 'namespace' \
    -xa '(kubie exec -e (__kubie_get_first_arg) default -- kubie ns 2>/dev/null)'

complete -c kubie -n "$cmd exec; and __kubie_at_arg 1" -a '"*"' -d 'exec in all contexts'
complete -c kubie -n "$cmd exec; and not __kubie_got_two_args" -l exit-early -e
complete -c kubie -n "$cmd exec; and not __kubie_got_two_args" -l context-headers \
    -xa "auto always never" -d 'print context header?'
complete -c kubie -n "$cmd exec; and __kubie_at_arg 2" -d 'namespace' \
    -a '(kubie exec -e (__kubie_get_first_arg) default -- kubie ns 2>/dev/null)'
complete -c kubie -n "$cmd exec; and __kubie_got_two_args" \
    -a '(__fish_complete_subcommand --commandline (__kubie_positionals)[4..-1])'

complete -c kubie -n "$cmd info" -a "ctx ns depth"

complete -c kubie -n "$cmd ns" -l unset -s u
complete -c kubie -n "$cmd ns" -d 'namespace' -a '(kubie ns 2>/dev/null)'
complete -c kubie -n "$cmd ns" -a '-' -d 'switch back'

complete -c kubie -n "$cmd export; and __kubie_at_arg 1" -d 'context' \
    -a '(kubie ctx 2>/dev/null)'

complete -c kubie -n "$cmd generate-completion" -l shell -d 'shell kind' \
    -xa "bash zsh fish xonsh nu"

# Strip the cmdline from options and flags, used for ctx and exec completions
function __kubie_positionals
    set -l cmd (commandline -poc)[2..-1] (commandline -ct)
    argparse r/recursive f/kubeconfig= n/namespace= e/exit-early c-context-headers= -- $cmd 2>&1
    for x in $argv; echo $x; end
end

function __kubie_get_first_arg
    # 2 because first elem is subcmd name
    echo (__kubie_positionals)[2]
end

function __kubie_at_arg
    test (count (__kubie_positionals)) = (math $argv[1] + 1)
end

function __kubie_got_two_args
    test (count (__kubie_positionals)) -ge 4
end
