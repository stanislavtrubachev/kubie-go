package kubie

import (
	"path/filepath"
	"testing"
)

func TestExpandUser(t *testing.T) {
	expected := filepath.Join(homeDir(), "hello/world/*.foo")
	actual := expandUser("~/hello/world/*.foo")
	if actual != expected {
		t.Errorf("expandUser(%q) = %q, want %q", "~/hello/world/*.foo", actual, expected)
	}
}
