package kubie

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"
)

// GetNamespaces получает список неймспейсов из кластера, используя kubectl.
// Если передан kubeconfig, он будет записан во временный файл и использован.
// Если kubeconfig == nil, используется переменная окружения KUBIE_KUBECONFIG.
func GetNamespaces(kubeconfig *KubeConfig) ([]string, error) {
	cmd := exec.Command("kubectl", "get", "namespaces")

	var tempFile *os.File
	if kubeconfig != nil {
		// Создаём временный файл
		f, err := os.CreateTemp("", "kubie-config-*.yaml")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		tempFile = f

		// Записываем конфиг во временный файл
		if err := kubeconfig.WriteToFile(tempFile.Name()); err != nil {
			tempFile.Close()
			os.Remove(tempFile.Name())
			return nil, err
		}
		tempFile.Close()

		// Устанавливаем KUBECONFIG
		cmd.Env = append(os.Environ(), "KUBECONFIG="+tempFile.Name())
		defer os.Remove(tempFile.Name())
	} else {
		// Проверяем переменную окружения KUBIE_KUBECONFIG
		kubeconfigPath, ok := os.LookupEnv("KUBIE_KUBECONFIG")
		if !ok {
			return nil, fmt.Errorf("KUBIE_KUBECONFIG variable is not set")
		}
		cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	}

	// Выполняем команду
	output, err := cmd.Output()
	if err != nil {
		// Проверяем, есть ли stderr
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := exitErr.Stderr
			// Пытаемся декодировать как UTF-8
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
