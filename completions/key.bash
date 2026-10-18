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
        COMPREPLY=( $(compgen -W "path list ls get copy cp info meta show describe set rm del delete unset note notes unnote backfill undo help" -- "$cur") )
        return
    fi

    local names
    names="$(key list 2>/dev/null)"

    case "$sub" in
        get|copy|cp|info|meta|show|describe)
            # exactly one key name
            [[ $COMP_CWORD -eq 2 ]] && COMPREPLY=( $(compgen -W "$names" -- "$cur") )
            ;;
        rm|del|delete|unset)
            # rm [-y] <name>
            if [[ $COMP_CWORD -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "-y --yes $names" -- "$cur") )
            elif [[ $COMP_CWORD -eq 3 && ( "${COMP_WORDS[2]}" == "-y" || "${COMP_WORDS[2]}" == "--yes" ) ]]; then
                COMPREPLY=( $(compgen -W "$names" -- "$cur") )
            fi
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
            # set name[=value] — offer existing names both bare (value from
            # stdin) and with a trailing '=' (inline value)
            COMPREPLY=( $(compgen -W "$names" -- "$cur") $(compgen -W "$(printf '%s\n' $names | sed 's/$/=/')" -- "$cur") )
            ;;
        list|ls)
            if [[ "$prev" == "--sort" ]]; then
                COMPREPLY=( $(compgen -W "name created modified" -- "$cur") )
            else
                COMPREPLY=( $(compgen -W "-l --long --all --sort" -- "$cur") )
            fi
            ;;
        backfill)
            COMPREPLY=( )
            ;;
    esac
}

complete -F _key_complete key
