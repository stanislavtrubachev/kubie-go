package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	kubie "github.com/stanislavtrubacev/kubie-go/kubielib"
	"github.com/stanislavtrubacev/kubie-go/kubielib/health"
	"github.com/stanislavtrubacev/kubie-go/shell"
)

// EnterContext processes the transition to the specified context, updates the session,
// checks the namespace (if necessary) and either overwrites the current kubeconfig or starts a new shell.
func EnterContext(settings *kubie.Settings, installed kubie.Installed, contextName string, namespaceName *string,
	recursive bool) error {

	state, err := kubie.Load()
	if err != nil {
		return err
	}
	var session kubie.Session
	sess, err := session.Load()
	if err != nil {
		return err
	}
	session = *sess

	var kubeconfig *kubie.KubeConfig
	if contextName == "-" {
		prev := session.GetLastContext()
		if prev != nil {
			// inside kubie-go shell using previous context from story
			ns := namespaceName
			if ns == nil && prev.Namespace != nil {
				ns = prev.Namespace
			}
			kc, err := installed.MakeKubeconfigForContext(prev.Context, ns)
			if err != nil {
				return err
			}
			kubeconfig = kc
		} else if state.LastContext != nil {

			// beyond kubie-go shell using latest global context
			var ns *string
			if namespaceName != nil {
				ns = namespaceName
			} else if nsFromHistory, ok := state.NamespaceHistory[*state.LastContext]; ok && nsFromHistory != nil {
				ns = nsFromHistory
			}
			kc, err := installed.MakeKubeconfigForContext(*state.LastContext, ns)
			if err != nil {
				return err
			}
			kubeconfig = kc
		} else {
			return fmt.Errorf("there is no previous context to switch to")
		}
	} else {
		// usual contex
		var ns *string
		if namespaceName != nil {
			ns = namespaceName
		} else if nsFromHistory, ok := state.NamespaceHistory[contextName]; ok && nsFromHistory != nil {
			ns = nsFromHistory
		}
		kc, err := installed.MakeKubeconfigForContext(contextName, ns)
		if err != nil {
			return err
		}
		kubeconfig = kc
	}

	if len(kubeconfig.Contexts) == 0 {
		return fmt.Errorf("generated kubeconfig has no contexts")
	}

	// save an entry in the session history
	ctxName := kubeconfig.Contexts[0].Name
	var ns *string
	if kubeconfig.Contexts[0].Context.Namespace != nil {
		ns = kubeconfig.Contexts[0].Context.Namespace
	}
	if err := session.RecordContextEntry(ctxName, ns); err != nil {
		return err
	}

	// check namespace
	if settings.Behavior.ValidateNamespaces.CanListNamespaces() {
		if namespaceName != nil {
			namespaces, err := kubie.GetNamespaces(kubeconfig)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not fetch namespace list: %v\n", err)
			} else {
				found := false
				for _, n := range namespaces {
					if n == *namespaceName {
						found = true
						break
					}
				}
				if !found {
					fmt.Fprintf(os.Stderr, "Warning: namespace %s does not exist.\n", *namespaceName)
				}
			}
		}
	}

	// Inside kubie-go shell and not recursive: just overwrite the kubeconfig in-place.
	if kubie.IsKubieActive() && !recursive {
		kubeconfigPath, err := kubie.GetKubeconfigPath()
		if err != nil {
			return err
		}
		if err := kubeconfig.WriteToFile(kubeconfigPath); err != nil {
			return err
		}
		if err := session.Save(""); err != nil {
			return err
		}
		return nil
	}

	// Spawning a new child shell: set up animated spinner in PS1.
	// Create a temp file that the shell daemon reads every 150ms for the current
	// spinner state. A goroutine runs QuickCheck concurrently and writes the final
	// status ("ok"/"warn"/"err") to the file when done.
	// Animation is disabled if: config says so, stdout is not a TTY, or TERM=dumb.
	animEnabled := !settings.Animation.Disable &&
		term.IsTerminal(int(os.Stdout.Fd())) &&
		os.Getenv("TERM") != "dumb"

	spinnerFile := ""
	if animEnabled {
		if f, err2 := os.CreateTemp("", "kubie-anim-*.txt"); err2 == nil {
			spinnerFile = f.Name()
			// Write an initial frame so the daemon has something to read on first tick.
			_, _ = f.WriteString("⠋:0")
			_ = f.Close()

			if kcBytes, err2 := yaml.Marshal(kubeconfig); err2 == nil {
				go func() {
					ctx2, cancel := context.WithTimeout(context.Background(), 8*time.Second)
					defer cancel()
					var status []byte
					switch health.QuickCheck(ctx2, kcBytes) {
					case health.StatusOK:
						status = []byte("ok")
					case health.StatusWarning:
						status = []byte("warn")
					default:
						status = []byte("err")
					}
					_ = os.WriteFile(spinnerFile, status, 0600)
				}()
			}
		}
	}
	if spinnerFile != "" {
		defer os.Remove(spinnerFile)
	}

	if err := shell.SpawnShell(settings, *kubeconfig, &session, spinnerFile); err != nil {
		return err
	}

	return nil
}

// Context processes command  "ctx" and do translation to the specified context.
// If no context name is specified, it represents an interactive selection via fzf.
// When the selection is canceled, it completes without error.
// If kubeconfigs does not work, it downloads content only from user files.
func Context(settings *kubie.Settings, contextName *string, namespaceName *string, kubeconfigs []string, recursive bool) error {
	var installed *kubie.Installed
	var err error

	if len(kubeconfigs) == 0 {
		installed, err = kubie.GetInstalledContexts(settings)

	} else {
		installed, err = kubie.GetKubeconfigsContexts(kubeconfigs)
	}
	if err != nil {
		return err
	}

	var name string
	if contextName != nil {
		name = *contextName
	} else {
		res, err := SelectOrListContext(&settings.Fzf, settings, installed)
		if err != nil {
			return err
		}
		switch v := res.(type) {
		case SelectResultSelected:
			name = v.Value
		default:
			return nil
		}
	}

	return EnterContext(settings, *installed, name, namespaceName, recursive)
}
