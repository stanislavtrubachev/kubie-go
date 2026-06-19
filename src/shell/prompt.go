package shell

import (
	"fmt"
	"os"
	"strings"

	kubie "github.com/stanislavtrubacev/kubie-go/kubielib"
)

type Command struct {
	Content   string
	ShellKind ShellKind
}

func NewCommand(content string, shellKind ShellKind) Command {
	return Command{
		Content:   content,
		ShellKind: shellKind, // type shell
	}
}

// String implements the fmt.Stringer interface for Command.
// For the Fish shell, the command is output as (command), for all other shells— as $(command).
func (c Command) String() string {
	switch c.ShellKind {
	case ShellKindFish:
		return "(" + c.Content + ")"
	default:
		return "$(" + c.Content + ")"
	}
}

func NewColor[D any](color uint32, content D, shellKind ShellKind) Color[D] {
	return Color[D]{
		Color:     color,
		Content:   content,
		ShellKind: shellKind,
	}
}

type Color[D any] struct {
	Color     uint32
	Content   D // D is the content type (it can be any, but fmt.Stringer or string conversion is used for formatting).
	ShellKind ShellKind
}

// isolate wraps the contents according to the type of shell.
// For Fish and Xonsh - unchanged, for Zsh - in %{...%}, for Bash - in \[...\].
func (c Color[D]) isolate(content string) string {
	switch c.ShellKind {
	case ShellKindFish, ShellKindXonsh:
		return content
	case ShellKindZsh:
		return "%{" + content + "%}"
	case ShellKindBash:
		return "\\[" + content + "\\]"
	default:
		return content
	}
}

// startColor returns an escape sequence for setting the color, wrapped in isolate according to the shell type.
func (c Color[D]) startColor(color uint32) string {
	var seq string
	if c.ShellKind == ShellKindXonsh {
		seq = fmt.Sprintf("\\033[%dm", color)
	} else {
		seq = fmt.Sprintf("\\e[%dm", color)
	}
	return c.isolate(seq)
}

// endColor returns an escape sequence for resetting the color, wrapped in isolate according to the shell type.
func (c Color[D]) endColor() string {
	var seq string
	if c.ShellKind == ShellKindXonsh {
		seq = "\\033[0m"
	} else {
		seq = "\\e[0m"
	}
	return c.isolate(seq)
}

// String implements fmt.Stringer for Color, returning a colored string.
// Format: [color] + content + [color reset], taking into account the shell features.
func (c Color[D]) String() string {
	return c.startColor(c.Color) + fmt.Sprint(c.Content) + c.endColor()
}

// ANSI color codes for terminal escape sequences.
const (
	RED   uint32 = 31
	GREEN uint32 = 32
	BLUE  uint32 = 34
)

// GeneratePS1 generates a PS1 string showing the current context, namespace, and depth.
// Takes into account the type of shell for the correct escaping of escape sequences.
func GeneratePS1(settings *kubie.Settings, depth uint32, shellKind ShellKind) string {
	exePath, err := os.Executable()
	if err != nil {
		panic("Could not get own binary path")
	}

	var parts []string

	ctxCmd := NewCommand(fmt.Sprintf("%s info ctx", exePath), shellKind)
	ctxColor := NewColor(RED, ctxCmd, shellKind)
	parts = append(parts, ctxColor.String())

	nsCmd := NewCommand(fmt.Sprintf("%s info ns", exePath), shellKind)
	nsColor := NewColor(GREEN, nsCmd, shellKind)
	parts = append(parts, nsColor.String())

	if settings.Prompt.ShowDepth && depth > 1 {
		depthColor := NewColor(BLUE, depth, shellKind)
		parts = append(parts, depthColor.String())
	}

	return "[" + strings.Join(parts, "|") + "]"
}
