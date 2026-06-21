// Package promptanim generates shell startup code (Bash and Zsh) that embeds
// an animated Braille spinner into the shell prompt (PS1).
//
// Zsh real-time animation uses zle -F (file-descriptor watcher):
//
//  1. A background daemon writes spinner state to a temp file every 150 ms
//     and writes one byte to a FIFO to wake zle.
//  2. zle -F registers the FIFO fd: when daemon writes, zle calls _kubie_redraw.
//  3. _kubie_redraw consumes the trigger byte, rebuilds PS1, calls zle reset-prompt.
//     zle reset-prompt redraws the prompt line WITHOUT losing the user's typed text.
//  4. precmd also rebuilds PS1 on each Enter so the state is always current.
//
// Bash has no zle equivalent — prompt updates on Enter only (PROMPT_COMMAND).
//
// Color mapping (zsh native sequences, immune to PROMPT_SUBST):
//
//	bright=0  → %F{8}     (ANSI 90, dark-gray)
//	bright=1  → %F{15}    (ANSI 97, light-gray)
//	ok        → %F{green}
//	warn      → %F{yellow}
//	err       → %F{red}
//	ctx name  → %F{red}
//	namespace → %F{green}
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
// Real-time animation via zle -F: the FIFO fd is watched by zle; every 150 ms
// the daemon writes one byte to wake zle, which calls zle reset-prompt.
// User input is preserved — zle reset-prompt saves and restores $BUFFER.
//
// This snippet is the sole owner of PS1.
func ZshCode(ctxName, ns, spinnerFile string) string {
	qFile := shellQuote(spinnerFile)
	qCtx := shellQuote(ctxName)
	qNs := shellQuote(ns)

	return fmt.Sprintf(`
# --- kubie animated prompt ---
__kubie_spin_file__=%s
__kubie_pipe__="${__kubie_spin_file__}.pipe"
__kubie_frames__=(⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)

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

# Shared PS1 builder used by both precmd and the zle widget.
# Uses native zsh %%F{}/%%f prompt sequences — immune to PROMPT_SUBST.
# Spinning:    entire block in one gray shade (dark or light)
# Final state: status-coloured ▮, ctx red, ns green
function _kubie_build_ps1() {
    local s=''
    [[ -f "$__kubie_spin_file__" ]] && s=$(<"$__kubie_spin_file__")
    local _ctx=%s _ns=%s
    local _sym _col _done=0
    case "$s" in
        ok)   _sym='▮'; _col='green';  _done=1 ;;
        warn) _sym='▮'; _col='yellow'; _done=1 ;;
        err)  _sym='▮'; _col='red';    _done=1 ;;
        *:1)  _sym="${s%%%%:*}"; _col='15' ;;
        *:0)  _sym="${s%%%%:*}"; _col='8'  ;;
        *)    _sym='▮'; _col='8'; _done=1 ;;
    esac
    if (( _done )); then
        if [[ -n "$_ns" ]]; then
            PS1="%%F{${_col}}[${_sym}%%f %%F{red}${_ctx}%%f%%F{white}|%%f%%F{green}${_ns}%%f] %%# "
        else
            PS1="%%F{${_col}}[${_sym}%%f %%F{red}${_ctx}%%f] %%# "
        fi
    else
        if [[ -n "$_ns" ]]; then
            PS1="%%F{${_col}}[${_sym} ${_ctx}|${_ns}%%f] %%# "
        else
            PS1="%%F{${_col}}[${_sym} ${_ctx}%%f] %%# "
        fi
    fi
}

# precmd: rebuild PS1 after each command (Enter key)
function __kubie_precmd__() {
    _kubie_build_ps1
}
add-zsh-hook precmd __kubie_precmd__

# zle widget: triggered by zle -F when daemon writes to FIFO.
# Consumes the trigger byte, rebuilds PS1, redraws prompt preserving user input.
function _kubie_redraw() {
    read -k1 -u $_kubie_fd 2>/dev/null
    _kubie_build_ps1
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
            PS1="\[\e[${_c}m\][${sym}\[\e[0m\] \[\e[31m\]${ctx}\[\e[0m\]\[\e[37m\]|\[\e[0m\]\[\e[32m\]${ns}\[\e[0m\]] \$ "
        else
            PS1="\[\e[${_c}m\][${sym}\[\e[0m\] \[\e[31m\]${ctx}\[\e[0m\]] \$ "
        fi
    else
        if [[ -n "$ns" ]]; then
            PS1="\[\e[${_c}m\][${sym} ${ctx}|${ns}\[\e[0m\]] \$ "
        else
            PS1="\[\e[${_c}m\][${sym} ${ctx}\[\e[0m\]] \$ "
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
