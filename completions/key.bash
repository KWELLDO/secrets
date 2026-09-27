# bash completion for `key` (the flat secrets CLI)
# Install: ~/.local/share/bash-completion/completions/key  (done by install.sh)
# Requires bash-completion (lazy-loaded on first `key <Tab>`).

_key_complete() {
    local cur prev sub
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    sub="${COMP_WORDS[1]}"

    # First word after `key`: the subcommand.
    if [[ $COMP_CWORD -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "path list ls get copy cp info meta show describe set rm del delete unset note notes unnote backfill help" -- "$cur") )
        return
    fi

    local names
    names="$(key list 2>/dev/null)"

    case "$sub" in
        get|copy|cp|info|meta|show|describe|rm|del|delete|unset)
            # exactly one key name
            [[ $COMP_CWORD -eq 2 ]] && COMPREPLY=( $(compgen -W "$names" -- "$cur") )
            ;;
        note|notes)
            # note <name> [label=value]
            [[ $COMP_CWORD -eq 2 ]] && COMPREPLY=( $(compgen -W "$names" -- "$cur") )
            ;;
        unnote)
            # unnote <name> <label>
            [[ $COMP_CWORD -eq 2 ]] && COMPREPLY=( $(compgen -W "$names" -- "$cur") )
            ;;
        set)
            # set name=value — offer existing names with a trailing '='
            COMPREPLY=( $(compgen -W "$(printf '%s\n' $names | sed 's/$/=/')" -- "$cur") )
            ;;
        list|ls)
            COMPREPLY=( $(compgen -W "-l --long --all" -- "$cur") )
            ;;
        backfill)
            COMPREPLY=( )
            ;;
    esac
}

complete -F _key_complete key
