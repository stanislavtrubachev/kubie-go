package kubie

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"
)

// GetNamespaces gets a list of namespaces from the cluster using kubectl.
// If kubeconfig is passed, it will be written to a temporary file and used.
// If kubeconfig is nil, the KUBIE_KUBECONFIG environment variable is used.
func GetNamespaces(kubeconfig *KubeConfig) ([]string, error) {
	cmd := exec.Command("kubectl", "get", "namespaces")

	var tempFile *os.File
	if kubeconfig != nil {
		f, err := os.CreateTemp("", "kubie-config-*.yaml")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		tempFile = f

		if err := kubeconfig.WriteToFile(tempFile.Name()); err != nil {
			tempFile.Close()
			os.Remove(tempFile.Name())
			return nil, err
		}
		tempFile.Close()

		cmd.Env = append(os.Environ(), "KUBECONFIG="+tempFile.Name())
		defer os.Remove(tempFile.Name())
	} else {
		kubeconfigPath, ok := os.LookupEnv("KUBIE_KUBECONFIG")
		if !ok {
			return nil, fmt.Errorf("KUBIE_KUBECONFIG variable is not set")
		}
		cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := exitErr.Stderr
			if !utf8.Valid(stderr) {
				stderrText := "could not decode stderr of kubectl as utf-8"
				return nil, fmt.Errorf("error calling kubectl:\n%s", stderrText)
			}
			return nil, fmt.Errorf("error calling kubectl:\n%s", string(stderr))
		}
		return nil, fmt.Errorf("failed to run kubectl: %w", err)
	}

	text := string(output)
	lines := strings.Split(text, "\n")
	var namespaces []string
	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}
		idx := strings.Index(line, " ")
		if idx == -1 {
			idx = len(line)
		}
		namespaces = append(namespaces, line[:idx])
	}
	return namespaces, nil
}
