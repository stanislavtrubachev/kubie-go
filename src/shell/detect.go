package shell

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode/utf8"
)

type ShellKind string

const (
	ShellKindBash  ShellKind = "bash"
	ShellKindFish  ShellKind = "fish"
	ShellKindXonsh ShellKind = "xonsh"
	ShellKindZsh   ShellKind = "zsh"
	ShellKindNu    ShellKind = "nu"
)

// ShellKindFromStr tries to identify ShellKind by the string name.
func ShellKindFromStr(name string) (ShellKind, bool) {
	switch name {
	case "bash", "dash":
		return ShellKindBash, true
	case "fish":
		return ShellKindFish, true
	case "xonsh", "python":
		return ShellKindXonsh, true
	case "zsh":
		return ShellKindZsh, true
	case "nu":
		return ShellKindNu, true
	default:
		return "", false
	}
}

// RunPs executes the ps command with the specified arguments and returns a list of non-empty output lines.
func RunPs(args []string) ([]string, error) {
	cmd := exec.Command("ps", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderrBytes := exitErr.Stderr
			var stderrStr string
			if utf8.Valid(stderrBytes) {
				stderrStr = string(stderrBytes)
			} else {
				stderrStr = "Could not decode stderr of ps as utf-8"
			}
			return nil, fmt.Errorf("error calling ps: %s", stderrStr)
		}
		return nil, fmt.Errorf("could not spawn ps: %w", err)
	}

	stdoutBytes := stdout.Bytes()
	if !utf8.Valid(stdoutBytes) {
		return nil, fmt.Errorf("ps output is not valid UTF-8")
	}
	text := string(stdoutBytes)
	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

// parentOf returns the PID of the parent process for the specified PID.
func ParentOf(pid string) (string, error) {
	lines, err := RunPs([]string{"-o", "ppid=", pid})
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("could not get parent pid of pid=%s", pid)
	}
	return strings.TrimSpace(lines[0]), nil
}

// CommandOf returns the process's command line by its PID.
func CommandOf(pid string) (string, error) {
	lines, err := RunPs([]string{"-o", "args=", pid})
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("could not get command of pid=%s", pid)
	}
	return lines[0], nil
}

// ParseCommand extracts the binary file name from the command line (removes the path, leading hyphens, ending digits, and dots.).
func ParseCommand(cmd string) string {
	// Найти первый пробел
	idx := strings.Index(cmd, " ")
	if idx == -1 {
		idx = len(cmd)
	}
	binaryPath := cmd[:idx]

	lastSlash := strings.LastIndex(binaryPath, "/")
	if lastSlash == -1 {
		lastSlash = 0
	} else {
		lastSlash += 1
	}
	binary := binaryPath[lastSlash:]

	binary = strings.TrimLeft(binary, "-")

	binary = strings.TrimRightFunc(binary, func(r rune) bool {
		return (r >= '0' && r <= '9') || r == '.'
	})

	return binary
}

// Detect determines which shell kubie-go was launched from.
// Goes up the process tree until it finds a known shell.
// Warning: Requires the ps command in the PATH.
func Detect() (ShellKind, error) {
	kubiePid := strconv.Itoa(os.Getpid())
	parentPid, err := ParentOf(kubiePid)
	if err != nil {
		return "", err
	}

	for {
		if parentPid == "1" {
			return "", fmt.Errorf("could not detect shell in use")
		}

		cmd, err := CommandOf(parentPid)
		if err != nil {
			return "", err
		}
		name := ParseCommand(cmd)
		if kind, ok := ShellKindFromStr(name); ok {
			return kind, nil
		}

		parentPid, err = ParentOf(parentPid)
		if err != nil {
			return "", err
		}
	}
}
