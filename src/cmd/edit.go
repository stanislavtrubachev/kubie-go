package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/stanislavtrubacev/kubie-go/kubielib"
)

// EditorCommand provides a command to launch a text editor.
// Contains the path to editor executable file and a list of command-line arguments.
type EditorCommand struct {
	Executable string
	Args       []string
}

// String implements fmt.Stringer, returning a string representation of the command.
// If there are no arguments, only the path to the executable file is returned,
// otherwise the path and arguments are separated by a space.
func (ec EditorCommand) String() string {
	if len(ec.Args) == 0 {
		return ec.Executable
	}
	return ec.Executable + " " + strings.Join(ec.Args, " ")
}

// ParseEditorCommand parses the line of the editor command, separated by spaces.
// The first token becomes the path to the executable file, the rest become arguments.
func ParseEditorCommand(raw string) (EditorCommand, error) {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return EditorCommand{}, fmt.Errorf("executable is empty")
	}
	return EditorCommand{
		Executable: parts[0],
		Args:       parts[1:],
	}, nil
}

// GetEditor defines the editor command that should be used
func GetEditor(settings *kubie.Settings) (EditorCommand, error) {

	if settings.DefaultEditor != nil && *settings.DefaultEditor != "" {
		cmd, err := ParseEditorCommand(*settings.DefaultEditor)
		if err != nil {
			return EditorCommand{}, fmt.Errorf("unable to parse default_editor command %s: %w", *settings.DefaultEditor, err)
		}
		return cmd, nil
	}

	if editorEnv := os.Getenv("EDITOR"); editorEnv != "" {
		cmd, err := ParseEditorCommand(editorEnv)
		if err != nil {
			return EditorCommand{}, fmt.Errorf("unable to parse EDITOR command %s: %w", editorEnv, err)
		}
		return cmd, nil
	}

	candidates := []string{"nvim", "vim", "emacs", "vi", "nano"}
	for _, name := range candidates {
		path, err := exec.LookPath(name)
		if err == nil {
			return EditorCommand{
				Executable: path,
				Args:       []string{},
			}, nil
		}
	}

	return EditorCommand{}, fmt.Errorf("could not find any editor to use")
}

// EditContext launches the editor to modify the kubeconfig file
// containing the specified context. If contextName is nil, the user is prompted to select the context via fzf
func EditContext(settings *kubie.Settings, contextName *string) error {
	installed, err := kubie.GetInstalledContexts(settings)
	if err != nil {
		return err
	}
	sort.Slice(installed.Contexts, func(i, j int) bool {
		return installed.Contexts[i].Item.Name < installed.Contexts[j].Item.Name
	})

	var selected string
	if contextName != nil {
		selected = *contextName
	} else {
		res, err := SelectOrListContext(&settings.Fzf, installed)
		if err != nil {
			return err
		}
		switch v := res.(type) {
		case SelectResultSelected:
			selected = v.Value
		default:
			return nil
		}
	}

	contextSrc := installed.FindContextByName(selected)
	if contextSrc == nil {
		return fmt.Errorf("could not find context %s", selected)
	}

	editorCmd, err := GetEditor(settings)
	if err != nil {
		return err
	}

	args := append(editorCmd.Args, contextSrc.Source)
	cmd := exec.Command(editorCmd.Executable, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to spawn editor command '%s': %w", editorCmd, err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	return nil
}

// EditConfig opens Kubie-go configuration file (~/.kube/kubie.yaml) in a text editor
func EditConfig(settings *kubie.Settings) error {
	editorCmd, err := GetEditor(settings)
	if err != nil {
		return err
	}

	settingsPath := kubie.SettingsPath()

	args := append(editorCmd.Args, settingsPath)
	cmd := exec.Command(editorCmd.Executable, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to spawn editor command '%s': %w", editorCmd, err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	return nil
}
