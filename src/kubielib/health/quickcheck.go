package health

import (
	"context"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// BuildClientsFromBytes creates a kubernetes Clientset from raw kubeconfig YAML bytes.
// This is used when a kubeconfig object is already in memory (no temp file needed).
func BuildClientsFromBytes(data []byte) (*kubernetes.Clientset, error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig(data)
	if err != nil {
		return nil, err
	}
	cfg.Timeout = 5 * time.Second
	return kubernetes.NewForConfig(cfg)
}

// QuickCheck performs a fast two-step cluster health check:
//  1. API server reachability
//  2. Node readiness
//
// Returns the worst CheckStatus found across both checks.
// StatusError means the cluster is unreachable or critically broken.
// StatusWarning means the cluster is reachable but has unhealthy nodes.
// StatusOK means everything looks fine.
func QuickCheck(ctx context.Context, kcBytes []byte) CheckStatus {
	k8s, err := BuildClientsFromBytes(kcBytes)
	if err != nil {
		return StatusError
	}

	var mu sync.Mutex
	worst := StatusOK

	upgrade := func(s CheckStatus) {
		mu.Lock()
		if s > worst {
			worst = s
		}
		mu.Unlock()
	}

	var wg sync.WaitGroup

	// Check 1: API server reachability.
	wg.Add(1)
	go func() {
		defer wg.Done()
		c := checkAPIServer(ctx, k8s)
		upgrade(c.Status)
	}()

	// Check 2: node readiness.
	wg.Add(1)
	go func() {
		defer wg.Done()
		nodes, err := k8s.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			upgrade(StatusWarning)
			return
		}
		h := &ClusterHealth{NodeSummary: NodeSummary{Total: len(nodes.Items)}}
		for _, n := range nodes.Items {
			info := buildNodeInfo(n)
			switch info.Status {
			case "Ready":
				h.NodeSummary.Ready++
			case "NotReady":
				h.NodeSummary.NotReady++
			default:
				h.NodeSummary.Unknown++
			}
		}
		c := checkNodesReady(h)
		upgrade(c.Status)
	}()

	wg.Wait()
	return worst
}
