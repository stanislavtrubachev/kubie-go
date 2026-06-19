package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/stanislavtrubacev/kubie-go/kubielib"
	"github.com/stanislavtrubacev/kubie-go/shell"
)

// Kubie all possible CLI commands and Kind type of commands:
// "ctx", "ns", "info", "exec", "export", "lint", "edit", "edit-config", "update", "delete", "generate-completion"
type Kubie struct {
	Kind string `json:"kind"`

	// ctx
	ContextName   *string  `json:"context_name,omitempty"`
	NamespaceName *string  `json:"namespace_name,omitempty"`
	Kubeconfigs   []string `json:"kubeconfigs,omitempty"`
	Recursive     bool     `json:"recursive,omitempty"`

	// ns
	Unset bool `json:"unset,omitempty"`

	// info
	InfoKind KubieInfoKind `json:"info_kind,omitempty"` // "ctx", "ns", "depth"

	// exec
	ExitEarly          bool                         `json:"exit_early,omitempty"`
	ContextHeadersFlag *kubie.ContextHeaderBehavior `json:"context_headers_flag,omitempty"`
	Args               []string                     `json:"args,omitempty"`

	// export
	// use ContextName and NamespaceName

	// edit and delete
	// use ContextName (optional)

	// generate-completion
	GenerateCompletion *GenerateCompletionCommand `json:"generate_completion,omitempty"`
}

// GenerateCompletionCommand and Shell it's optional shell name (if not specified, it is determined automatically)
type GenerateCompletionCommand struct {
	Shell *shell.ShellKind `yaml:"shell,omitempty"`
}

// GenerateCompletion generates an auto-completion script for the specified shell
// TODO:implement it after enabling the CLI framework (cobra)?
func GenerateCompletion(command GenerateCompletionCommand) {
	_ = determineShell(command)
	fmt.Fprintln(os.Stderr, "generate-completion: not yet implemented")
	os.Exit(1)
}

// determineShell defines the shell type for generating auto-completion
func determineShell(command GenerateCompletionCommand) shell.ShellKind {
	if command.Shell != nil {
		return *command.Shell
	}
	if shell := shellFromEnv(); shell != "" {
		return shell
	}
	fmt.Fprintln(os.Stderr, "Could not determine shell from environment. Please specify the shell.")
	os.Exit(1)
	return ""
}

// shellFromEnv returns the shell type based on the SHELL environment variable
// shell type: bash, zsh, fish, xonsh, nu
func shellFromEnv() shell.ShellKind {
	shellEnv := os.Getenv("SHELL")
	if shellEnv == "" {
		return ""
	}
	// get name from path (example: /bin/bash -> bash)
	parts := strings.Split(shellEnv, "/")
	name := parts[len(parts)-1]
	if kind, ok := shell.ShellKindFromStr(name); ok {
		return kind
	}
	return ""
}
