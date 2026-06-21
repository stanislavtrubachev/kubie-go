package health

import (
	"context"
	"testing"
)

func TestBuildClientsFromBytes_invalidYAML(t *testing.T) {
	_, err := BuildClientsFromBytes([]byte("not: valid: kubeconfig: yaml:::"))
	if err == nil {
		t.Error("BuildClientsFromBytes should fail on invalid kubeconfig YAML")
	}
}

func TestBuildClientsFromBytes_empty(t *testing.T) {
	_, err := BuildClientsFromBytes([]byte(""))
	if err == nil {
		t.Error("BuildClientsFromBytes should fail on empty data")
	}
}

func TestBuildClientsFromBytes_validStructure(t *testing.T) {
	// Minimal valid kubeconfig structure (won't connect, but should parse).
	yaml := `
apiVersion: v1
kind: Config
clusters:
  - name: test
    cluster:
      server: https://localhost:6443
contexts:
  - name: test
    context:
      cluster: test
      user: test
users:
  - name: test
    user: {}
current-context: test
`
	k8s, err := BuildClientsFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("BuildClientsFromBytes should succeed on valid structure: %v", err)
	}
	if k8s == nil {
		t.Error("returned clientset should not be nil")
	}
}

func TestQuickCheck_invalidBytes(t *testing.T) {
	status := QuickCheck(context.Background(), []byte("bad-data"))
	if status != StatusError {
		t.Errorf("QuickCheck with invalid bytes should return StatusError, got %d", status)
	}
}

func TestQuickCheck_cancelledContext(t *testing.T) {
	// Use a valid-structure kubeconfig pointing to a non-existent server.
	yaml := `
apiVersion: v1
kind: Config
clusters:
  - name: test
    cluster:
      server: https://127.0.0.1:19999
contexts:
  - name: test
    context:
      cluster: test
      user: test
users:
  - name: test
    user: {}
current-context: test
`
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	status := QuickCheck(ctx, []byte(yaml))
	// With a cancelled context or unreachable server, we expect non-OK.
	if status == StatusOK {
		t.Error("QuickCheck against unreachable server with cancelled context should not return StatusOK")
	}
}
