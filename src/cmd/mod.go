package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/stanislavtrubacev/kubie-go/kubielib"
	"golang.org/x/term"
)

// SelectResult an interface that combines possible selection results
type SelectResult interface {
	// to ensure that only our types implement interface
	isSelectResult()
}

// SelectResultCancelled user canceled the selection
type SelectResultCancelled struct{}

func (SelectResultCancelled) isSelectResult() {}

// SelectResultListed user viewed the list (output to stdout)
type SelectResultListed struct{}

func (SelectResultListed) isSelectResult() {}

// SelectResultSelected user has selected the context
type SelectResultSelected struct {
	Value string
}

func (SelectResultSelected) isSelectResult() {}

// SelectOrListContext selects the context interactively (via fzf) or displays a list
func SelectOrListContext(fzf *kubie.Fzf, installed *kubie.Installed) (SelectResult, error) {
	sort.Slice(installed.Contexts, func(i, j int) bool {
		return installed.Contexts[i].Item.Name < installed.Contexts[j].Item.Name
	})

	contextNames := make([]string, len(installed.Contexts))
	for i, c := range installed.Contexts {
		contextNames[i] = c.Item.Name
	}

	if len(contextNames) == 0 {
		return nil, fmt.Errorf("no contexts found")
	}
	if len(contextNames) == 1 {
		return SelectResultSelected{Value: contextNames[0]}, nil
	}

	if term.IsTerminal(int(os.Stdout.Fd())) {
		reversed := make([]string, len(contextNames)) // contexts are flipped for skim
		for i, name := range contextNames {
			reversed[len(contextNames)-1-i] = name
		}

		selected, err := kubie.Select(fzf, reversed)
		if err != nil {
			return nil, err
		}
		if selected != "" {
			return SelectResultSelected{Value: selected}, nil
		}
		return SelectResultCancelled{}, nil
	} else {
		for _, name := range contextNames {
			fmt.Println(name)
		}
		return SelectResultListed{}, nil
	}
}

// SelectOrListNamespace returns the result of namespace selection.
// If namespaces (not nil) is passed, it uses them, otherwise it gets the list via kubectl
func SelectOrListNamespace(fzf *kubie.Fzf, namespaces []string) (SelectResult, error) {
	var ns []string
	if namespaces != nil {
		ns = namespaces
	} else {
		var err error
		ns, err = kubie.GetNamespaces(nil)
		if err != nil {
			return nil, fmt.Errorf("could not get namespaces: %w", err)
		}
	}

	sort.Strings(ns)

	if len(ns) == 0 {
		return nil, fmt.Errorf("no namespaces found")
	}

	if term.IsTerminal(int(os.Stdout.Fd())) {
		reversed := make([]string, len(ns)) // contexts are flipped for skim
		for i, name := range ns {
			reversed[len(ns)-1-i] = name
		}

		selected, err := kubie.Select(fzf, reversed)
		if err != nil {
			return nil, err
		}
		if selected != "" {
			return SelectResultSelected{Value: selected}, nil
		}
		return SelectResultCancelled{}, nil
	} else {
		for _, name := range ns {
			fmt.Println(name)
		}
		return SelectResultListed{}, nil
	}
}
