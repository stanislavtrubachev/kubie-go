package kubie

import (
	"encoding/json"
	"fmt"
	"os"
)

// Session contains information limited to kubie-go session.
// Currently stores the history of contexts and namespaces,
// so that users can return to the previous context using `-`
type Session struct {
	History []HistoryEntry `json:"history"`
}

type HistoryEntry struct {
	Context   string  `json:"context"`
	Namespace *string `json:"namespace,omitempty"`
}

// Load downloads a session from a file
func (s *Session) Load() (*Session, error) {
	path := GetSessionPath()
	if path == "" {
		return &Session{}, nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Session{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}

	return &sess, nil
}

// Save - saves the session to a file. If path is not an empty string, this path is used.
// Otherwise, the value of the KUBIE_SESSION environment variable is used via GetSessionPath()
func (s *Session) Save(path string) error {
	sessionPath := path
	if sessionPath == "" {
		sessionPath = GetSessionPath()
		if sessionPath == "" {
			return fmt.Errorf("KUBIE_SESSION env variable missing")
		}
	}

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(sessionPath, data, 0644)
}

// AddHistoryEntry adds a new entry to the session history
func (s *Session) AddHistoryEntry(context string, namespace *string) {
	entry := HistoryEntry{
		Context: context,
	}
	// todo: need refract
	if namespace != nil {
		entry.Namespace = namespace
	}
	s.History = append(s.History, entry)
}

// Global state, single-threaded usage is assumed
// todo: `sync.Mutex` is better for thread safety
var globalState = &State{
	NamespaceHistory: make(map[string]*string),
}

// RecordContextEntry adds an entry to session history and updates the global status.
// Namespace is saved to the namespace history for this context. The current context is set as the last one used
func (s *Session) RecordContextEntry(contextName string, namespace *string) error {
	s.AddHistoryEntry(contextName, namespace)

	if namespace != nil {
		globalState.NamespaceHistory[contextName] = namespace
	}
	globalState.LastContext = &contextName

	return nil
}

// GetLastContext returns the last used context, which is different from the current (or last) context in history
func (s *Session) GetLastContext() *HistoryEntry {
	if len(s.History) == 0 {
		return nil
	}
	currentContext := s.History[len(s.History)-1].Context
	for i := len(s.History) - 2; i >= 0; i-- {
		if s.History[i].Context != currentContext {
			return &s.History[i]
		}
	}
	return nil
}

// GetLastNamespace returns the last used namespace in the current context, which is different from current namespace
func (s *Session) GetLastNamespace() *string {
	if len(s.History) == 0 {
		return nil
	}
	current := s.History[len(s.History)-1]
	for i := len(s.History) - 2; i >= 0; i-- {
		entry := s.History[i]
		// If the context differs from the current one, we interrupt the search.
		if current.Context != entry.Context {
			return nil
		}

		// Checking if the namespace is different
		if (current.Namespace == nil && entry.Namespace != nil) ||
			(current.Namespace != nil && entry.Namespace == nil) ||
			(current.Namespace != nil && entry.Namespace != nil && *current.Namespace != *entry.Namespace) {
			return entry.Namespace
		}
	}
	return nil
}
