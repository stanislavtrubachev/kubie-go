package shell

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/stanislavtrubacev/kubie-go/kubielib/health/promptanim"
)

// SpawnShellZsh launches an interactive zsh shell with KUBECONFIG pre-installed and a custom prompt.
// Creates a temporary directory with .zshrc, installs ZDOTDIR, executes the start/stop hooks.
func SpawnShellZsh(info *ShellSpawnInfo) error {
	dir, err := os.MkdirTemp("", "kubie-zsh-")
	if err != nil {
		return fmt.Errorf("could not create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	zshrcPath := filepath.Join(dir, ".zshrc")
	zshrcFile, err := os.Create(zshrcPath)
	if err != nil {
		return fmt.Errorf("could not open zshrc file: %w", err)
	}
	defer zshrcFile.Close()

	writer := bufio.NewWriter(zshrcFile)

	rcContent := `
# Reference for loading behavior
# https://shreevatsa.wordpress.com/2008/03/30/zshbash-startup-files-loading-order-bashrc-zshrc-etc/

if [[ -f "/etc/zshenv" ]] ; then
    source "/etc/zshenv"
elif [[ -f "/etc/zsh/zshenv" ]] ; then
    source "/etc/zsh/zshenv"
fi

if [[ -f "$HOME/.zshenv" ]] ; then
    tmp_ZDOTDIR=$ZDOTDIR
    source "$HOME/.zshenv"
    # If the user has overridden $ZDOTDIR, we save that in $_KUBIE_USER_ZDOTDIR for later reference
    # and reset $ZDOTDIR
    if [[ "$tmp_ZDOTDIR" != "$ZDOTDIR" ]]; then
        _KUBIE_USER_ZDOTDIR=$ZDOTDIR
        ZDOTDIR=$tmp_ZDOTDIR
        unset tmp_ZDOTDIR
    fi
fi

# If a zsh_history file exists, copy it over before zsh initialization so history is maintained
if [[ -f "$HOME/.zsh_history" ]] ; then
    cp $HOME/.zsh_history $ZDOTDIR
fi

KUBIE_LOGIN_SHELL=0
if [[ "$OSTYPE" == "darwin"* ]] ; then
    KUBIE_LOGIN_SHELL=1
fi

if [[ -f "/etc/zprofile" && "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    source "/etc/zprofile"
elif [[ -f "/etc/zsh/zprofile" && "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    source "/etc/zsh/zprofile"
fi

if [[ -f "${_KUBIE_USER_ZDOTDIR:-$HOME}/.zprofile" && "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    source "${_KUBIE_USER_ZDOTDIR:-$HOME}/.zprofile"
fi

if [[ -f "/etc/zshrc" ]] ; then
    source "/etc/zshrc"
elif [[ -f "/etc/zsh/zshrc" ]] ; then
    source "/etc/zsh/zshrc"
fi

if [[ -f "${_KUBIE_USER_ZDOTDIR:-$HOME}/.zshrc" ]] ; then
    ZDOTDIR=$_KUBIE_USER_ZDOTDIR \
        source "${_KUBIE_USER_ZDOTDIR:-$HOME}/.zshrc"
fi

if [[ -f "/etc/zlogin" && "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    source "/etc/zlogin"
elif [[ -f "/etc/zsh/zlogin" && "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    source "/etc/zsh/zlogin"
fi

if [[ -f "${_KUBIE_USER_ZDOTDIR:-$HOME}/.zlogin" && "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    source "${_KUBIE_USER_ZDOTDIR:-$HOME}/.zlogin"
fi

unset _KUBIE_USER_ZDOTDIR

autoload -Uz add-zsh-hook

# This function sets the proper KUBECONFIG variable before a command runs,
# in case something overwrote it.
function __kubie_cmd_pre_exec__() {
    export KUBECONFIG="$KUBIE_KUBECONFIG"
}

add-zsh-hook preexec __kubie_cmd_pre_exec__
`
	if _, err := writer.WriteString(rcContent); err != nil {
		return fmt.Errorf("failed to write rc content: %w", err)
	}
	
	// When animation is active it owns PS1 completely, so skip the standard
	// kubie prompt hook to avoid two precmd functions fighting over PS1.
	animActive := info.SpinnerFile != "" && !info.Settings.Animation.Disable
	if !info.Settings.Prompt.Disable && !animActive {
		promptSection := fmt.Sprintf(`
# Activate prompt substitution.
setopt PROMPT_SUBST

# This function fixes the prompt via a precmd hook.
function __kubie_cmd_pre_cmd__() {
    local KUBIE_PROMPT=$'%s'

    # If KUBIE_ZSH_USE_RPS1 is set, we use RPS1 instead of PS1.
    if [[ "$KUBIE_ZSH_USE_RPS1" == "1" ]] ; then

        # Avoid modifying RPS1 again if the RPS1 has not been reset.
        if [[ "$RPS1" != *"$KUBIE_PROMPT"* ]] ; then

            # If RPS1 is empty, we do not seperate with a space.
            if [[ -z "$RPS1" ]] ; then
                RPS1="$KUBIE_PROMPT"
            else
                RPS1="$KUBIE_PROMPT $RPS1"
            fi
        fi
    else
        # Avoid modifying PS1 again if the PS1 has not been reset.
        if [[ "$PS1" != *"$KUBIE_PROMPT"* ]] ; then
            PS1="$KUBIE_PROMPT $PS1"
        fi
    fi
}

# When promptinit is activated, a precmd hook which updates PS1 is installed.
# In order to inject the kubie PS1 when promptinit is activated, we must
# also add our own precmd hook which modifies PS1 after promptinit themes.
add-zsh-hook precmd __kubie_cmd_pre_cmd__
`, info.Prompt)
		if _, err := writer.WriteString(promptSection); err != nil {
			return fmt.Errorf("failed to write prompt: %w", err)
		}
	}

	if animActive {
		animCode := promptanim.ZshCode(info.CtxName, info.NS, info.SpinnerFile)
		if _, err := writer.WriteString(animCode + "\n"); err != nil {
			return fmt.Errorf("failed to write spinner code: %w", err)
		}
	}

	if info.Settings.Hooks.StartCtx != "" {
		if _, err := writer.WriteString(info.Settings.Hooks.StartCtx + "\n"); err != nil {
			return fmt.Errorf("failed to write start hook: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush zshrc: %w", err)
	}
	zshrcFile.Close()

	cmd := exec.Command("zsh")
	overrides := make(map[string]string, len(info.EnvVars.Vars)+1)
	for k, v := range info.EnvVars.Vars {
		overrides[k] = v
	}
	overrides["ZDOTDIR"] = dir
	cmd.Env = MergeEnv(overrides)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("zsh execution failed: %w", err)
	}

	if info.Settings.Hooks.StopCtx != "" {
		stopFile, err := os.CreateTemp("", "kubie-zsh-exit-hook-*.zsh")
		if err != nil {
			return fmt.Errorf("failed to create stop hook temp file: %w", err)
		}
		defer os.Remove(stopFile.Name())
		defer stopFile.Close()

		if _, err := stopFile.WriteString(info.Settings.Hooks.StopCtx); err != nil {
			return fmt.Errorf("failed to write stop hook: %w", err)
		}
		if err := stopFile.Close(); err != nil {
			return fmt.Errorf("failed to close stop hook file: %w", err)
		}

		stopCmd := exec.Command("zsh", stopFile.Name())
		stopCmd.Env = os.Environ()
		for k, v := range info.EnvVars.Vars {
			stopCmd.Env = append(stopCmd.Env, k+"="+v)
		}
		stopCmd.Stdin = os.Stdin
		stopCmd.Stdout = os.Stdout
		stopCmd.Stderr = os.Stderr

		if err := stopCmd.Run(); err != nil {
			return fmt.Errorf("stop hook execution failed: %w", err)
		}
	}

	return nil
}
