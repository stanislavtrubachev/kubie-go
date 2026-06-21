// Package promptanim generates shell startup code (Bash and Zsh) that embeds
// an animated Braille spinner into the shell prompt (PS1).
//
// Zsh strategy — prepend, don't replace:
//
//   - We insert a "[⠼ ctx|ns] " block at the START of the existing user PS1
//     instead of overwriting it entirely. The user's theme (oh-my-zsh, prezto,
//     etc.) continues to render git status, PWD, colours, etc.
//
// Real-time animation via zle -F:
//
//  1. A background daemon writes spinner state to a temp file every 150 ms
//     and writes one byte to a FIFO to wake zle.
//  2. zle -F registers the FIFO fd: when daemon writes, zle calls _kubie_redraw.
//  3. _kubie_redraw rebuilds the prefix, prepends to saved orig PS1, calls
//     zle reset-prompt — redraws prompt WITHOUT losing user's typed text.
//  4. precmd (runs LAST) saves the theme's PS1 each cycle, then prepends.
//
// Static-PS1 guard: __kubie_last_ps1__ detects when PS1 hasn't been refreshed
// by a theme precmd, so the prefix is never double-added.
//
// Bash: no zle — prompt updates on Enter only (PROMPT_COMMAND).
package promptanim

import (
	"fmt"
	"strings"
)

// shellQuote wraps s in single quotes, escaping any embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ZshCode returns the zsh snippet appended to the kubie .zshrc.
//
// The snippet prepends an animated "[⠼ ctx|ns] " block to the existing PS1
// so themes like oh-my-zsh or prezto continue to show git status and PWD.
func ZshCode(ctxName, ns, spinnerFile string) string {
	qFile := shellQuote(spinnerFile)
	qCtx := shellQuote(ctxName)
	qNs := shellQuote(ns)

	return fmt.Sprintf(`
# --- kubie animated prompt ---
__kubie_spin_file__=%s
__kubie_pipe__="${__kubie_spin_file__}.pipe"
__kubie_frames__=(⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)
__kubie_ctx__=%s
__kubie_ns__=%s
__kubie_orig_ps1__=''
__kubie_applied_prefix__=''
__kubie_prefix__=''

# FIFO: daemon writes one byte every 150ms → zle -F wakes → _kubie_redraw runs
mkfifo "$__kubie_pipe__"
exec {_kubie_fd}<>"$__kubie_pipe__"

# Background daemon: updates spinner file and writes one byte to FIFO per frame.
# Stops when Go writes "ok"/"warn"/"err" to the spinner file (sends final byte).
function __kubie_daemon__() {
    local idx=0
    while [[ -f "$__kubie_spin_file__" ]]; do
        local s
        s=$(<"$__kubie_spin_file__") 2>/dev/null
        if [[ "$s" == ok || "$s" == warn || "$s" == err ]]; then
            printf 'x' >&$_kubie_fd
            return 0
        fi
        local char="${__kubie_frames__[$((idx+1))]}"
        local bright=$(( (idx / 2) %% 2 ))
        printf '%%s:%%d' "$char" "$bright" > "$__kubie_spin_file__"
        printf 'x' >&$_kubie_fd
        idx=$(( (idx + 1) %% 10 ))
        sleep 0.15
    done
}

__kubie_daemon__ >/dev/null 2>&1 &!
__kubie_daemon_pid__=$!

# Builds __kubie_prefix__ — the "[⠼ ctx|ns] " block only (no %%# suffix).
# This prefix is PREPENDED to the existing user PS1, not a full replacement.
# Uses native zsh %%F{}/%%f prompt sequences, immune to PROMPT_SUBST issues.
function _kubie_build_prefix() {
    local s=''
    [[ -f "$__kubie_spin_file__" ]] && s=$(<"$__kubie_spin_file__")
    local _sym _col _done=0
    case "$s" in
        ok)   _sym='✓'; _col='green';  _done=1 ;;
        warn) _sym='⚠'; _col='yellow'; _done=1 ;;
        err)  _sym='✗'; _col='red';    _done=1 ;;
        *:1)  _sym="${s%%%%:*}"; _col='15' ;;
        *:0)  _sym="${s%%%%:*}"; _col='8'  ;;
        *)    _sym='▮'; _col='8'; _done=1 ;;
    esac
    if (( _done )); then
        if [[ -n "$__kubie_ns__" ]]; then
            __kubie_prefix__="[%%F{${_col}}${_sym}%%f %%F{red}${__kubie_ctx__}%%f%%F{white}|%%f%%F{green}${__kubie_ns__}%%f] "
        else
            __kubie_prefix__="[%%F{${_col}}${_sym}%%f %%F{red}${__kubie_ctx__}%%f] "
        fi
    else
        if [[ -n "$__kubie_ns__" ]]; then
            __kubie_prefix__="[%%F{${_col}}${_sym} ${__kubie_ctx__}|${__kubie_ns__}%%f] "
        else
            __kubie_prefix__="[%%F{${_col}}${_sym} ${__kubie_ctx__}%%f] "
        fi
    fi
}

# precmd: runs LAST (add-zsh-hook appends) so theme hooks have already set PS1.
# Compares current PS1 against the exact value we last built
# (applied_prefix + orig). If they differ, the theme changed PS1 — save
# the new value as the fresh orig. If they match, PS1 is ours — reuse orig.
# This prevents double-prefix both for static PS1 users and after _kubie_redraw
# updated PS1 between two Enter keystrokes.
function __kubie_precmd__() {
    if [[ "$PS1" != "${__kubie_applied_prefix__}${__kubie_orig_ps1__}" ]]; then
        __kubie_orig_ps1__="$PS1"
    fi
    _kubie_build_prefix
    PS1="${__kubie_prefix__}${__kubie_orig_ps1__}"
    __kubie_applied_prefix__="$__kubie_prefix__"
}
add-zsh-hook precmd __kubie_precmd__

# zle widget: triggered by zle -F when daemon writes to FIFO.
# Rebuilds prefix, reuses saved orig PS1, redraws prompt preserving user input.
# Must also update __kubie_applied_prefix__ so precmd knows which prefix we set.
function _kubie_redraw() {
    read -k1 -u $_kubie_fd 2>/dev/null
    _kubie_build_prefix
    PS1="${__kubie_prefix__}${__kubie_orig_ps1__}"
    __kubie_applied_prefix__="$__kubie_prefix__"
    zle reset-prompt
}
zle -N _kubie_redraw
zle -F $_kubie_fd _kubie_redraw

function __kubie_cleanup__() {
    zle -F $_kubie_fd 2>/dev/null
    exec {_kubie_fd}>&- 2>/dev/null
    kill "$__kubie_daemon_pid__" 2>/dev/null
    rm -f "$__kubie_spin_file__" "$__kubie_pipe__"
}
trap '__kubie_cleanup__' EXIT
# --- end kubie animated prompt ---
`, qFile, qCtx, qNs)
}

// BashCode returns the bash snippet injected into the kubie rcfile.
//
// Bash has no zle equivalent — prompt updates on Enter only via PROMPT_COMMAND.
func BashCode(ctxName, ns, spinnerFile string) string {
	qFile := shellQuote(spinnerFile)
	qCtx := shellQuote(ctxName)
	qNs := shellQuote(ns)

	return fmt.Sprintf(`
# --- kubie animated prompt ---
__kubie_spin_file__=%s
__kubie_frames__=(⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)

function __kubie_daemon__() {
    local idx=0
    while [[ -f "$__kubie_spin_file__" ]]; do
        local s
        s=$(<"$__kubie_spin_file__") 2>/dev/null
        [[ "$s" == ok || "$s" == warn || "$s" == err ]] && return 0
        local char="${__kubie_frames__[$idx]}"
        local bright=$(( (idx / 2) %% 2 ))
        printf '%%s:%%d' "$char" "$bright" > "$__kubie_spin_file__"
        idx=$(( (idx + 1) %% 10 ))
        sleep 0.15
    done
}

__kubie_daemon__ >/dev/null 2>&1 &
__kubie_daemon_pid__=$!
disown

function __kubie_build_ps1__() {
    local s=''
    [[ -f "$__kubie_spin_file__" ]] && s=$(<"$__kubie_spin_file__")

    local sym _c _done=0
    case "$s" in
        ok)   sym='▮'; _c=32; _done=1 ;;
        warn) sym='▮'; _c=33; _done=1 ;;
        err)  sym='▮'; _c=31; _done=1 ;;
        *:1)  sym="${s%%%%:*}"; _c=97 ;;
        *:0)  sym="${s%%%%:*}"; _c=90 ;;
        *)    sym='▮'; _c=90; _done=1 ;;
    esac

    local ctx=%s ns=%s
    if (( _done )); then
        if [[ -n "$ns" ]]; then
            PS1="[\[\e[${_c}m\]${sym}\[\e[0m\] \[\e[31m\]${ctx}\[\e[0m\]\[\e[37m\]|\[\e[0m\]\[\e[32m\]${ns}\[\e[0m\]] \$ "
        else
            PS1="[\[\e[${_c}m\]${sym}\[\e[0m\] \[\e[31m\]${ctx}\[\e[0m\]] \$ "
        fi
    else
        if [[ -n "$ns" ]]; then
            PS1="[\[\e[${_c}m\]${sym} ${ctx}|${ns}\[\e[0m\]] \$ "
        else
            PS1="[\[\e[${_c}m\]${sym} ${ctx}\[\e[0m\]] \$ "
        fi
    fi
}

PROMPT_COMMAND='__kubie_build_ps1__'

function __kubie_cleanup__() {
    kill "$__kubie_daemon_pid__" 2>/dev/null
    rm -f "$__kubie_spin_file__"
}
trap '__kubie_cleanup__' EXIT
# --- end kubie animated prompt ---
`, qFile, qCtx, qNs)
}
