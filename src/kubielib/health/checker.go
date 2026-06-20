package health

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// RunChecks performs all health checks and appends results to ClusterHealth.
func RunChecks(ctx context.Context, h *ClusterHealth, k8s *kubernetes.Clientset, mc *metricsv.Clientset) {
	h.Checks = append(h.Checks, checkAPIServer(ctx, k8s))
	h.Checks = append(h.Checks, checkCoreDNS(ctx, k8s))
	h.Checks = append(h.Checks, checkMetricsServer(ctx, mc))
	h.Checks = append(h.Checks, checkNodesReady(h))
	h.Checks = append(h.Checks, checkKubeSystemPods(ctx, k8s))
}

func checkAPIServer(ctx context.Context, k8s *kubernetes.Clientset) HealthCheck {
	_, err := k8s.Discovery().ServerVersion()
	if err != nil {
		return HealthCheck{Name: "API Server", Status: StatusError, Detail: err.Error()}
	}
	return HealthCheck{Name: "API Server", Status: StatusOK, Detail: "reachable"}
}

func checkCoreDNS(ctx context.Context, k8s *kubernetes.Clientset) HealthCheck {
	pods, err := k8s.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
		LabelSelector: "k8s-app=kube-dns",
	})
	if err != nil {
		return HealthCheck{Name: "CoreDNS", Status: StatusError, Detail: err.Error()}
	}
	running := 0
	for _, p := range pods.Items {
		if p.Status.Phase == "Running" {
			running++
		}
	}
	if running == 0 {
		return HealthCheck{Name: "CoreDNS", Status: StatusError, Detail: "no running pods found"}
	}
	return HealthCheck{Name: "CoreDNS", Status: StatusOK, Detail: fmt.Sprintf("%d pod(s) running", running)}
}

func checkMetricsServer(ctx context.Context, mc *metricsv.Clientset) HealthCheck {
	if mc == nil {
		return HealthCheck{Name: "Metrics Server", Status: StatusWarning, Detail: "client not initialized"}
	}
	_, err := mc.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return HealthCheck{Name: "Metrics Server", Status: StatusWarning, Detail: "not available (metrics will be skipped)"}
	}
	return HealthCheck{Name: "Metrics Server", Status: StatusOK, Detail: "available"}
}

func checkNodesReady(h *ClusterHealth) HealthCheck {
	if h.NodeSummary.Total == 0 {
		return HealthCheck{Name: "Nodes Ready", Status: StatusWarning, Detail: "no nodes found"}
	}
	if h.NodeSummary.NotReady > 0 || h.NodeSummary.Unknown > 0 {
		return HealthCheck{
			Name:   "Nodes Ready",
			Status: StatusWarning,
			Detail: fmt.Sprintf("%d/%d ready", h.NodeSummary.Ready, h.NodeSummary.Total),
		}
	}
	return HealthCheck{
		Name:   "Nodes Ready",
		Status: StatusOK,
		Detail: fmt.Sprintf("all %d node(s) ready", h.NodeSummary.Total),
	}
}

func checkKubeSystemPods(ctx context.Context, k8s *kubernetes.Clientset) HealthCheck {
	pods, err := k8s.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return HealthCheck{Name: "kube-system pods", Status: StatusError, Detail: err.Error()}
	}
	bad := 0
	for _, p := range pods.Items {
		if p.Status.Phase != "Running" && p.Status.Phase != "Succeeded" {
			bad++
		}
	}
	if bad > 0 {
		return HealthCheck{
			Name:   "kube-system pods",
			Status: StatusWarning,
			Detail: fmt.Sprintf("%d pod(s) not running", bad),
		}
	}
	return HealthCheck{Name: "kube-system pods", Status: StatusOK, Detail: "all running"}
}
