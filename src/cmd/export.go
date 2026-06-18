package cmd

import (
	"fmt"
	"os"

	"github.com/stanislavtrubacev/kubie-go/kubielib"
)

// Export creates temporary kubeconfig files for all contexts matching the template and outputs their paths to stdout
func Export(settings *kubie.Settings, contextName, namespaceName string) error {
	installed, err := kubie.GetInstalledContexts(settings)
	if err != nil {
		return err
	}

	matching := installed.GetContextsMatching(contextName, settings.Behavior.AllowMultipleContextPatterns)
	if len(matching) == 0 {
		return fmt.Errorf("no context matching %s", contextName)
	}

	for _, contextSrc := range matching {
		kubeconfig, err := installed.MakeKubeconfigForContext(contextSrc.Item.Name, &namespaceName)
		if err != nil {
			return err
		}

		tmpFile, err := os.CreateTemp("", "kubie-config-*.yaml")
		if err != nil {
			return err
		}
		tmpFile.Close()

		if err := kubeconfig.WriteToFile(tmpFile.Name()); err != nil {
			return err
		}
		fmt.Println(tmpFile.Name())
	}

	os.Exit(0)
	return nil
}
