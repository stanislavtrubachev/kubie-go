package shell

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
)

// SpawnShellXonsh launches an interactive xonsh shell with KUBECONFIG preinstalled and a custom prompt.
// Creates a temporary rc file, passes it via --rc, executes hooks, and shuts down.
func SpawnShellXonsh(info *ShellSpawnInfo) error {

	rcFile, err := os.CreateTemp("", "kubie-xonshrc-*.xsh")
	if err != nil {
		return fmt.Errorf("could not create temp rc file: %w", err)
	}
	defer os.Remove(rcFile.Name())
	defer rcFile.Close()

	writer := bufio.NewWriter(rcFile)

	rcContent := `
# https://xon.sh/xonshrc.html
from pathlib import Path

files = [
    "/etc/xonshrc",
    "~/.xonshrc",
    "~/.config/xonsh/rc.xsh",
]
for file in files:
    if Path(file).is_file():
        source @(file)
if Path("~/.config/xonsh/rc.d").is_dir():
    for file in path.glob('*.xsh'):
        source @(file)

@events.on_precommand
def __kubie_cmd_pre_exec__(cmd):
    $KUBECONFIG = $KUBIE_KUBECONFIG
`
	if _, err := writer.WriteString(rcContent); err != nil {
		return fmt.Errorf("failed to write rc content: %w", err)
	}

	if !info.Settings.Prompt.Disable {
		promptSection := fmt.Sprintf(`
$KUBIE_PROMPT='%s'
import re

# Fanciful prompt-command replacement as xonsh forces the use of PROMPT_FIELDS
for match in re.finditer(r'\$\(([^)]*)\)', $KUBIE_PROMPT):
    command = match.group(1)
    name = command.split().pop()
    $PROMPT_FIELDS[name] = evalx(f'lambda: $({{command}}).strip()')
    $KUBIE_PROMPT = $KUBIE_PROMPT.replace(f'$({{command}})', '{{' + name + '}}')

if $KUBIE_XONSH_USE_RIGHT_PROMPT == "1":
    $RIGHT_PROMPT = $KUBIE_PROMPT + $RIGHT_PROMPT
else:
    $PROMPT = $KUBIE_PROMPT + $PROMPT

del $KUBIE_PROMPT
`, info.Prompt)
		if _, err := writer.WriteString(promptSection); err != nil {
			return fmt.Errorf("failed to write prompt: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush rc file: %w", err)
	}
	rcFile.Close()

	cmd := exec.Command("xonsh", "--rc", rcFile.Name())
	cmd.Env = MergeEnv(info.EnvVars.Vars)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xonsh execution failed: %w", err)
	}

	return nil
}
