package shell

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"

	kubie "github.com/stanislavtrubacev/kubie-go/kubielib"
)

type ShellSpawnInfo struct {
	Settings *kubie.Settings
	Prompt   string
	EnvVars  EnvVars
}

// SpawnShellBash launches a new bash shell with the passed parameters.
// Creates a temporary rc file with the settings, executes it and when exiting performs a stop hook, if specified.
func SpawnShellBash(info *ShellSpawnInfo) error {
	// 1. Создаём временный rc-файл
	tempRcFile, err := os.CreateTemp("", "kubie-bashrc-*.bash")
	if err != nil {
		return fmt.Errorf("failed to create temp rc file: %w", err)
	}
	defer os.Remove(tempRcFile.Name())
	defer tempRcFile.Close()

	writer := bufio.NewWriter(tempRcFile)

	rcContent := `
KUBIE_LOGIN_SHELL=0
if [[ "$OSTYPE" == "darwin"* ]] ; then
    KUBIE_LOGIN_SHELL=1
fi

# Reference for loading behavior
# https://shreevatsa.wordpress.com/2008/03/30/zshbash-startup-files-loading-order-bashrc-zshrc-etc/

if [[ "$KUBIE_LOGIN_SHELL" == "1" ]] ; then
    if [[ -f "/etc/profile" ]] ; then
        source "/etc/profile"
    fi

    if [[ -f "$HOME/.bash_profile" ]] ; then
        source "$HOME/.bash_profile"
    elif [[ -f "$HOME/.bash_login" ]] ; then
        source "$HOME/.bash_login"
    elif [[ -f "$HOME/.profile" ]] ; then
        source "$HOME/.profile"
    fi
else
    if [[ -f "/etc/bash.bashrc" ]] ; then
        source "/etc/bash.bashrc"
    fi

    if [[ -f "$HOME/.bashrc" ]] ; then
        source "$HOME/.bashrc"
    fi
fi

function __kubie_cmd_pre_exec__() {
    export KUBECONFIG="$KUBIE_KUBECONFIG"
}

trap '__kubie_cmd_pre_exec__' DEBUG
`
	if _, err := writer.WriteString(rcContent); err != nil {
		return fmt.Errorf("failed to write rc content: %w", err)
	}

	if !info.Settings.Prompt.Disable {
		promptLine := fmt.Sprintf(`
KUBIE_PROMPT='%s'
PS1="$KUBIE_PROMPT $PS1"
unset KUBIE_PROMPT
`, info.Prompt)
		if _, err := writer.WriteString(promptLine); err != nil {
			return fmt.Errorf("failed to write prompt: %w", err)
		}
	}

	if info.Settings.Hooks.StartCtx != "" {
		if _, err := writer.WriteString(info.Settings.Hooks.StartCtx); err != nil {
			return fmt.Errorf("failed to write start hook: %w", err)
		}

		if _, err := writer.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush rc file: %w", err)
	}
	if err := tempRcFile.Close(); err != nil {
		return fmt.Errorf("failed to close rc file: %w", err)
	}

	cmd := exec.Command("bash", "--rcfile", tempRcFile.Name())
	cmd.Env = MergeEnv(info.EnvVars.Vars)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bash execution failed: %w", err)
	}

	if info.Settings.Hooks.StopCtx != "" {
		stopHookFile, err := os.CreateTemp("", "kubie-bash-exit-hook-*.bash")
		if err != nil {
			return fmt.Errorf("failed to create stop hook temp file: %w", err)
		}
		defer os.Remove(stopHookFile.Name())
		defer stopHookFile.Close()

		if _, err := stopHookFile.WriteString(info.Settings.Hooks.StopCtx); err != nil {
			return fmt.Errorf("failed to write stop hook: %w", err)
		}
		if err := stopHookFile.Close(); err != nil {
			return fmt.Errorf("failed to close stop hook file: %w", err)
		}

		stopCmd := exec.Command("bash", stopHookFile.Name())
		stopCmd.Env = MergeEnv(info.EnvVars.Vars)
		stopCmd.Stdin = os.Stdin
		stopCmd.Stdout = os.Stdout
		stopCmd.Stderr = os.Stderr

		if err := stopCmd.Run(); err != nil {
			return fmt.Errorf("stop hook execution failed: %w", err)
		}
	}

	return nil
}
