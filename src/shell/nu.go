package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SpawnShellNu it launches an interactive Nu shell (nushell) with preset environment variables and possibly a configurable prompt.
func SpawnShellNu(info *ShellSpawnInfo) error {

	var args string

	for name, value := range info.EnvVars.Vars {

		args += fmt.Sprintf("$env.%s = '%s';", name, value)

		if name == "KUBIE_PROMPT_DISABLE" && value == "0" {
			prompt := info.Prompt
			prompt = replaceAll(prompt, "\\[\\e[31m\\]", "")
			prompt = replaceAll(prompt, "\\[\\e[32m\\]", "")
			prompt = replaceAll(prompt, "\\[\\e[0m\\]", "")
			prompt = replaceAll(prompt, "$", "")

			promptCmd := fmt.Sprintf(`$env.PROMPT_COMMAND = { || $"%s\n(create_left_prompt)" };`, prompt)
			args += promptCmd
		}
	}

	cmd := exec.Command("nu", "-e", args)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nu execution failed: %w", err)
	}

	return nil
}

func replaceAll(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}
