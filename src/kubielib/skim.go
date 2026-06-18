package kubie

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// SkimOptions settings for skim/fzf.
type SkimOptions struct {
	NoMulti bool
	NoMouse bool
	Reverse bool
	Color   *string
	Case    string
	NoInfo  bool
	Height  *string
	Prompt  *string
}

// BuildOptions creates SkimOptions based on the Fzf configuration
// Deprecated: rewrite, not use
func BuildOptions(fzf Fzf) (*SkimOptions, error) {
	opts := &SkimOptions{
		NoMulti: true,
		NoMouse: !fzf.Mouse,
		Reverse: fzf.Reverse,
	}
	if fzf.Color != nil {
		opts.Color = fzf.Color
	}
	if fzf.IgnoreCase {
		opts.Case = "ignore"
	}
	if fzf.InfoHidden {
		opts.NoInfo = true
	}
	if fzf.Height != nil {
		opts.Height = fzf.Height
	}
	if fzf.Prompt != nil {
		opts.Prompt = fzf.Prompt
	}
	return opts, nil
}

const (
	maxVisible = 15
	ansiReset  = "\033[0m"
	ansiRev    = "\033[7m"
	eraseLine  = "\033[K"
)

// selectInteractive shows an arrow-key navigable list on stderr.
// Enter confirms selection; ESC or Ctrl-C cancels.
func selectInteractive(items []string) (string, error) {
	if len(items) == 0 {
		return "", nil
	}
	if len(items) == 1 {
		return items[0], nil
	}

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return selectFallback(items)
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return selectFallback(items)
	}
	defer term.Restore(fd, oldState)

	w := os.Stderr
	visible := maxVisible
	if len(items) < visible {
		visible = len(items)
	}

	selected := 0
	offset := 0

	render := func() {
		for i := 0; i < visible; i++ {
			idx := offset + i
			var line string
			if idx < len(items) {
				if idx == selected {
					line = fmt.Sprintf("\r%s> %s%s%s\r\n", ansiRev, items[idx], ansiReset, eraseLine)
				} else {
					line = fmt.Sprintf("\r  %s%s\r\n", items[idx], eraseLine)
				}
			} else {
				line = fmt.Sprintf("\r%s\r\n", eraseLine)
			}
			fmt.Fprint(w, line)
		}

		fmt.Fprintf(w, "\033[%dA", visible) // Move cursor back to top of the rendered block
	}

	clear := func() {
		// Move down past all rendered lines, then clear upward
		fmt.Fprintf(w, "\033[%dB", visible)
		for i := 0; i < visible; i++ {
			fmt.Fprintf(w, "\r%s\033[A", eraseLine)
		}
		fmt.Fprintf(w, "\r%s", eraseLine)
	}

	render()

	// In raw mode arrow keys arrive as a 3-byte burst: ESC '[' A/B.
	// Using Read with a small buffer captures the full sequence in one call.
	buf := make([]byte, 4)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			clear()
			return "", nil
		}
		key := buf[:n]

		switch {
		case n == 1 && (key[0] == '\r' || key[0] == '\n'):
			clear()
			return items[selected], nil

		case n == 1 && (key[0] == 3 || key[0] == 4 || key[0] == 27): // Ctrl-C, Ctrl-D, ESC
			clear()
			return "", nil

		case n >= 3 && bytes.Equal(key[:3], []byte{27, '[', 'A'}): // Up arrow
			if selected > 0 {
				selected--
				if selected < offset {
					offset = selected
				}
				render()
			}

		case n >= 3 && bytes.Equal(key[:3], []byte{27, '[', 'B'}): // Down arrow
			if selected < len(items)-1 {
				selected++
				if selected >= offset+visible {
					offset = selected - visible + 1
				}
				render()
			}
		}
	}
}

// selectFallback is used when stdin is not a terminal (e.g. piped input).
func selectFallback(items []string) (string, error) {
	for _, item := range items {
		fmt.Println(item)
	}
	return "", nil
}

// Select runs an interactive selector. Uses fzf if available, otherwise uses
// the built-in arrow-key selector.
func Select(fzf *Fzf, items []string) (string, error) {
	if _, err := exec.LookPath("fzf"); err == nil {
		return selectWithFzf(fzf, items)
	}
	return selectInteractive(items)
}

func selectWithFzf(fzf *Fzf, items []string) (string, error) {
	args := []string{}
	if !fzf.Mouse {
		args = append(args, "--no-mouse")
	}
	if fzf.Reverse {
		args = append(args, "--reverse")
	}
	if fzf.IgnoreCase {
		args = append(args, "--ignore-case")
	}
	if fzf.InfoHidden {
		args = append(args, "--no-info")
	}
	if fzf.Height != nil {
		args = append(args, "--height", *fzf.Height)
	}
	if fzf.Prompt != nil {
		args = append(args, "--prompt", *fzf.Prompt)
	}
	if fzf.Color != nil {
		args = append(args, "--color", *fzf.Color)
	}

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n"))
	cmd.Stderr = os.Stderr

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}
