package path

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	once          sync.Once
	dataDir       string
	statePath     string
	stateLockPath string
)

// initPaths initializes paths once upon the first call
func initPaths() {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		panic("Could not get local data dir")
	}
	dataDir = filepath.Join(baseDir, "kubie")
	statePath = filepath.Join(dataDir, "state.json")
	stateLockPath = filepath.Join(dataDir, ".state.json.lock")
}

// DataDir returns the path to the Kubie data directory
func DataDir() string {
	once.Do(initPaths)
	return dataDir
}

// State returns the path to the status file
func State() string {
	once.Do(initPaths)
	return statePath
}

// StateLock returns the path to the status lock file
// Deprecated: rewrite, not use now
func StateLock() string {
	once.Do(initPaths)
	return stateLockPath
}
