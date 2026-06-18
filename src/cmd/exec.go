package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/stanislavtrubacev/kubie-go/kubielib"
)

// RunInContext runs the passed command with a temporary kubeconfig,
// sets environment variables and forwards all signals to the child process.
// Returns the child process termination code or an error.
func RunInContext(kubeconfig *kubie.KubeConfig, args []string) (int, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("no command specified")
	}

	tmpFile, err := os.CreateTemp("", "kubie-config-*.yaml")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if err := kubeconfig.WriteToFile(tmpFile.Name()); err != nil {
		return 0, fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	depth := kubie.GetDepth()
	nextDepth := depth + 1

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGWINCH,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
	)

	// Start child process
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(os.Environ(),
		"KUBECONFIG="+tmpFile.Name(),
		"KUBIE_KUBECONFIG="+tmpFile.Name(),
		"KUBIE_ACTIVE=1",
		"KUBIE_DEPTH="+strconv.Itoa(int(nextDepth)),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		signal.Stop(sigChan)
		close(sigChan)
		return 0, fmt.Errorf("failed to start command: %w", err)
	}
	childPid := cmd.Process.Pid

	// for sending signals
	done := make(chan struct{})
	go func() {
		defer close(done)
		for sig := range sigChan {
			sysSig, ok := sig.(syscall.Signal)
			if !ok {
				continue
			}
			_ = syscall.Kill(childPid, sysSig) // ignore the error (so that process can end)
		}
	}()

	err = cmd.Wait()
	signal.Stop(sigChan)
	close(sigChan)
	<-done

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), nil
			}
			return 0, fmt.Errorf("failed to get exit status: %w", err)
		}
		return 0, fmt.Errorf("command execution failed: %w", err)
	}
	return 0, nil
}

// Exec executes the command with the specified context (or several if allow_multiple_context_patterns is enabled),
// If exit_early is true, terminates the process at the first non-zero return code.
// The context can be a template with '*' and '?'.
// After all contexts are executed, the process ends with code 0.
func Exec(settings *kubie.Settings, contextName string, namespaceName string, exitEarly bool, contextHeadersFlag *kubie.ContextHeaderBehavior, args []string) error {
	if len(args) == 0 {
		return nil
	}

	installed, err := kubie.GetInstalledContexts(settings)
	if err != nil {
		return err
	}

	matching := installed.GetContextsMatching(contextName, settings.Behavior.AllowMultipleContextPatterns)
	if len(matching) == 0 {
		return fmt.Errorf("no context matching %s", contextName)
	}

	printContext := false
	if contextHeadersFlag != nil {
		printContext = contextHeadersFlag.ShouldPrintHeaders()
	} else {
		printContext = settings.Behavior.PrintContextInExec.ShouldPrintHeaders()
	}

	for _, contextSrc := range matching {
		if printContext {
			fmt.Printf("CONTEXT => %s\n", contextSrc.Item.Name)
		}

		kubeconfig, err := installed.MakeKubeconfigForContext(contextSrc.Item.Name, &namespaceName)
		if err != nil {
			return err
		}

		returnCode, err := RunInContext(kubeconfig, args)
		if err != nil {
			return err
		}

		if printContext {
			fmt.Println(strings.Repeat("-", 20))
		}

		if returnCode != 0 && exitEarly {
			os.Exit(returnCode)
		}
	}

	os.Exit(0)
	return nil
}
