#compdef kubie

_kubie() {
    local state line
    typeset -A opt_args

    _arguments -C \
        '1: :_kubie_commands' \
        '*:: :->args'

    case $state in
        args)
            case $line[1] in
                ctx)
                    _arguments \
                        '(-r --recursive)'{-r,--recursive}'[push context onto existing kubie shell]' \
                        '(-n --namespace)'{-n,--namespace}'[namespace to set]:namespace:_kubie_namespaces' \
                        '1::context:_kubie_contexts'
                    ;;
                ns)
                    _arguments \
                        '(-r --recursive)'{-r,--recursive}'[push namespace onto existing kubie shell]' \
                        '--unset[unset namespace back to default]' \
                        '1::namespace:_kubie_namespaces'
                    ;;
                edit|delete)
                    _arguments '1::context:_kubie_contexts'
                    ;;
                exec)
                    _arguments \
                        '--exit-early[exit if kubectl returns non-zero]' \
                        '--context-headers[print context header]:behavior:(auto always never)' \
                        '1:context:_kubie_contexts' \
                        '2:namespace:_kubie_namespaces' \
                        '*:: :->kubectl_args'
                    ;;
                export)
                    _arguments \
                        '1:context:_kubie_contexts' \
                        '2:namespace:_kubie_namespaces'
                    ;;
                info)
                    _arguments '1:kind:(ctx ns depth)'
                    ;;
                generate-completion)
                    _arguments '--shell[shell kind]:shell:(bash zsh fish xonsh nu)'
                    ;;
                edit-config|lint|update)
                    # No further completions
                    ;;
            esac
            ;;
    esac
}

_kubie_commands() {
    local -a commands
    commands=(
        'ctx:switch kubernetes context'
        'ns:switch namespace in current context'
        'info:print current context, namespace, or depth'
        'exec:execute command in given context and namespace'
        'lint:check kubeconfigs for errors'
        'edit:edit a kubeconfig file'
        'edit-config:edit kubie configuration file'
        'delete:delete a context from kubeconfig'
        'export:export context to stdout'
        'update:update kubie to the latest version'
        'generate-completion:generate shell completion script'
    )
    _describe 'command' commands
}

_kubie_contexts() {
    local -a contexts
    contexts=(${(f)"$(kubie ctx 2>/dev/null)"})
    _describe 'context' contexts
}

_kubie_namespaces() {
    local -a namespaces
    namespaces=(${(f)"$(kubie ns 2>/dev/null)"})
    _describe 'namespace' namespaces
}

_kubie
