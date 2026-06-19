package kubie

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"gopkg.in/yaml.v3"
)

// ReadJSON reads JSON from a file using the specified path and deserializes it into a type T object
// Deprecated: rewrite, not use now
func ReadJSON[T any](path string) (T, error) {
	file, err := os.Open(path)
	if err != nil {
		var zero T
		return zero, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var obj T
	if err := json.NewDecoder(reader).Decode(&obj); err != nil {
		var zero T
		return zero, err
	}
	return obj, nil
}

// WriteJson writes the object to a JSON file, creating parent directories if necessary.
// Deprecated: rewrite, not use now
func WriteJson[T any](path string, obj *T) error {

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(obj); err != nil {
		return err
	}
	return nil
}

// ReadYaml reads YAML from a file using the specified path and deserializes it into a type T object.
// Deprecated: rewrite, not use now
func ReadYaml[T any](path string) (T, error) {
	file, err := os.Open(path)
	if err != nil {
		var zero T
		return zero, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var obj T
	dec := yaml.NewDecoder(reader)
	if err := dec.Decode(&obj); err != nil {
		var zero T
		return zero, err
	}
	return obj, nil
}

// WriteYaml writes the object to a file in YAML format, creating parent directories if necessary.
func WriteYaml[T any](path string, obj *T) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	enc := yaml.NewEncoder(writer)
	defer enc.Close()

	if err := enc.Encode(obj); err != nil {
		return err
	}
	return nil
}

// FileLock performs the scope function inside the file lock.
// The lock is captured exclusively on the path file.
// If there is a panic inside the scope, the lock is lifted and the panic is pushed on.
// Deprecated: rewrite, not use now
func FileLock[T any](path string, scope func() (T, error)) (T, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("could not open lock file at %s: %w", path, err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		var zero T
		return zero, fmt.Errorf("could not lock file at %s: %w", path, err)
	}
	defer func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	}()

	return scope()
}
