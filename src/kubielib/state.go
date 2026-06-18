package kubie

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/stanislavtrubacev/kubie-go/path"
)

// State global state for kubie-go
type State struct {
	LastContext      *string            `json:"last_context,omitempty"`
	NamespaceHistory map[string]*string `json:"namespace_history"`
}

// Load loads the status from a shared-lock file
func Load() (*State, error) {
	if err := os.MkdirAll(path.DataDir(), 0755); err != nil {
		return nil, fmt.Errorf("could not create data dir: %w", err)
	}

	stateFilePath := path.State()
	f, err := os.OpenFile(stateFilePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open state file: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH); err != nil {
		return nil, fmt.Errorf("failed to lock state file: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	if len(data) == 0 {
		return &State{NamespaceHistory: make(map[string]*string)}, nil
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}
	if state.NamespaceHistory == nil {
		state.NamespaceHistory = make(map[string]*string)
	}
	return &state, nil
}

// Modify loads the exclusive-lock state, passes it to modFn,
// then saves the changes back to the same file descriptor.
func Modify(modFn func(*State) error) error {
	if err := os.MkdirAll(path.DataDir(), 0755); err != nil {
		return fmt.Errorf("could not create data dir: %w", err)
	}

	stateFilePath := path.State()
	f, err := os.OpenFile(stateFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open state file: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock state file: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	var state State
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() > 0 {
		if _, err := f.Seek(0, 0); err != nil {
			return err
		}
		if err := json.NewDecoder(f).Decode(&state); err != nil {
			return fmt.Errorf("failed to decode state: %w", err)
		}
	} else {
		state.NamespaceHistory = make(map[string]*string)
	}

	if err := modFn(&state); err != nil {
		return err
	}

	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(&state)
}
