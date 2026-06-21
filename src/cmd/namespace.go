package cmd

import (
	"fmt"
	"strings"

	kubie "github.com/stanislavtrubacev/kubie-go/kubielib"
	"github.com/stanislavtrubacev/kubie-go/shell"
)

// Namespace processes "ns" command, name change in the current session
func Namespace(settings *kubie.Settings, namespaceName *string, recursive bool, unset bool) error {

	if err := kubie.EnsureKubieActive(); err != nil {
		return err
	}

	var sess kubie.Session
	session, err := sess.Load()
	if err != nil {
		return fmt.Errorf("could not load session file: %w", err)
	}

	// resetting the current namespace
	if namespaceName == nil && unset {
		return enterNamespace(settings, session, recursive, nil)
	}

	var actualNs *string
	if namespaceName != nil {
		s := *namespaceName

		// special value "-", switch to the previous one
		if s == "-" {
			lastNs := session.GetLastNamespace()
			if lastNs == nil {
				return fmt.Errorf("there is no previous namespace to switch to")
			}
			actualNs = lastNs
		} else {

			switch settings.Behavior.ValidateNamespaces {
			case kubie.ValidateNamespacesBehaviorFalse:
				actualNs = &s

			case kubie.ValidateNamespacesBehaviorTrue:
				namespaces, err := kubie.GetNamespaces(nil)
				if err != nil {
					return err
				}
				found := false
				for _, ns := range namespaces {
					if ns == s {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("'%s' is not a valid namespace for the context", s)
				}
				actualNs = &s

			case kubie.ValidateNamespacesBehaviorPartial:
				namespaces, err := kubie.GetNamespaces(nil)
				if err != nil {
					return err
				}
				exactMatch := false
				for _, ns := range namespaces {
					if ns == s {
						exactMatch = true
						break
					}
				}
				if exactMatch {
					actualNs = &s
				} else {
					// collecting partial matches (contain the substring s)
					var partialMatches []string
					for _, ns := range namespaces {
						if strings.Contains(ns, s) {
							partialMatches = append(partialMatches, ns)
						}
					}
					switch len(partialMatches) {
					case 0:
						return fmt.Errorf("'%s' is not a valid namespace for the context", s)
					case 1:
						actualNs = &partialMatches[0]
					default:
						// A few partial matches, suggest choose
						res, err := SelectOrListNamespace(&settings.Fzf, partialMatches)
						if err != nil {
							return err
						}
						switch v := res.(type) {
						case SelectResultSelected:
							actualNs = &v.Value
						default:
							return nil
						}
					}
				}

			default:
				return fmt.Errorf("unknown ValidateNamespacesBehavior")
			}
		}
	} else {
		res, err := SelectOrListNamespace(&settings.Fzf, nil)
		if err != nil {
			return err
		}
		switch v := res.(type) {
		case SelectResultSelected:
			actualNs = &v.Value
		default:
			return nil
		}
	}
	return enterNamespace(settings, session, recursive, actualNs)
}

// enterNamespace applies specified namespace to current context, updates
// state and history for session and then either starts a new shell (if recursive) or overwrites current kubeconfig.
func enterNamespace(settings *kubie.Settings, session *kubie.Session, recursive bool, namespaceName *string) error {
	config, err := kubie.GetCurrentConfig()
	if err != nil {
		return err
	}

	if len(config.Contexts) == 0 {
		return fmt.Errorf("no contexts in current kubeconfig")
	}
	config.Contexts[0].Context.Namespace = namespaceName

	contextName := config.Contexts[0].Name

	// updating the global state
	err = kubie.Modify(func(state *kubie.State) error {
		state.NamespaceHistory[contextName] = namespaceName
		return nil
	})
	if err != nil {
		return err
	}

	session.AddHistoryEntry(contextName, namespaceName)

	// launch a new shell, or overwrite the config
	if recursive {
		return shell.SpawnShell(settings, *config, session, "")
	} else {
		configPath, err := kubie.GetKubeconfigPath()
		if err != nil {
			return err
		}
		if err := config.WriteToFile(configPath); err != nil {
			return err
		}
		return session.Save("")
	}
}
