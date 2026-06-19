package cmd

import (
	"github.com/stanislavtrubacev/kubie-go/kubielib"
)

// DeleteContext deletes the specified context from all kubeconfig files.
// If contextName is nil, it offers to select the context via fzf.
func DeleteContext(settings *kubie.Settings, contextName *string) error {
	installed, err := kubie.GetInstalledContexts(settings)
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

	return installed.DeleteContext(name)
}
