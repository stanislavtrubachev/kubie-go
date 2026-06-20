package health

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// BuildClients creates a kubernetes and metrics client from the active kubeconfig.
func BuildClients() (*kubernetes.Clientset, *metricsv.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBIE_KUBECONFIG")
	}

	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("could not build kubeconfig: %w", err)
	}
	cfg.Timeout = 5 * time.Second

	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create k8s client: %w", err)
	}

	metrics, _ := metricsv.NewForConfig(cfg) // nil metrics client is handled gracefully
	return k8s, metrics, nil
}

// Collect gathers all cluster health data in parallel and returns ClusterHealth.
func Collect(ctx context.Context, k8s *kubernetes.Clientset, mc *metricsv.Clientset, namespace string) (*ClusterHealth, error) {
	h := &ClusterHealth{CollectedAt: time.Now()}

	// determine current context name
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBIE_KUBECONFIG")
	}
	if cfg, err := clientcmd.LoadFromFile(kubeconfig); err == nil && cfg.CurrentContext != "" {
		h.Context = cfg.CurrentContext
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := make([]string, 0)

	addErr := func(e error) {
		mu.Lock()
		errs = append(errs, e.Error())
		mu.Unlock()
	}

	// --- API version ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		if sv, err := k8s.Discovery().ServerVersion(); err == nil {
			mu.Lock()
			h.APIVersion = sv.GitVersion
			mu.Unlock()
		} else {
			addErr(fmt.Errorf("api version: %w", err))
		}
	}()

	// --- Nodes ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		nodes, err := k8s.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			addErr(fmt.Errorf("nodes: %w", err))
			return
		}
		mu.Lock()
		defer mu.Unlock()
		h.NodeSummary.Total = len(nodes.Items)
		for _, n := range nodes.Items {
			info := buildNodeInfo(n)
			h.Nodes = append(h.Nodes, info)
			switch info.Status {
			case "Ready":
				h.NodeSummary.Ready++
			case "NotReady":
				h.NodeSummary.NotReady++
			default:
				h.NodeSummary.Unknown++
			}
		}
	}()

	// --- Pods ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		listNS := namespace
		if listNS == "" {
			listNS = metav1.NamespaceAll
		}
		pods, err := k8s.CoreV1().Pods(listNS).List(ctx, metav1.ListOptions{})
		if err != nil {
			addErr(fmt.Errorf("pods: %w", err))
			return
		}
		mu.Lock()
		defer mu.Unlock()
		h.PodSummary.Total = len(pods.Items)
		for _, p := range pods.Items {
			switch p.Status.Phase {
			case corev1.PodRunning:
				h.PodSummary.Running++
			case corev1.PodPending:
				h.PodSummary.Pending++
			case corev1.PodFailed:
				h.PodSummary.Failed++
			case corev1.PodSucceeded:
				h.PodSummary.Succeeded++
			default:
				h.PodSummary.Unknown++
			}
			if reason, bad := podProblemReason(p); bad {
				switch reason {
				case "CrashLoopBackOff":
					h.PodSummary.CrashLoopBackOff++
				case "ImagePullBackOff", "ErrImagePull":
					h.PodSummary.ImagePullBackOff++
				}
				h.Pods = append(h.Pods, PodInfo{
					Namespace: p.Namespace,
					Name:      p.Name,
					Status:    string(p.Status.Phase),
					Reason:    reason,
				})
			}
		}
	}()

	// --- Namespaces ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		nsList, err := k8s.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			addErr(fmt.Errorf("namespaces: %w", err))
			return
		}
		mu.Lock()
		h.NamespaceCount = len(nsList.Items)
		mu.Unlock()
	}()

	// --- PVCs ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		listNS := namespace
		if listNS == "" {
			listNS = metav1.NamespaceAll
		}
		pvcs, err := k8s.CoreV1().PersistentVolumeClaims(listNS).List(ctx, metav1.ListOptions{})
		if err != nil {
			return // non-fatal
		}
		mu.Lock()
		defer mu.Unlock()
		for _, pvc := range pvcs.Items {
			h.PVCStatus = append(h.PVCStatus, PVCInfo{
				Namespace: pvc.Namespace,
				Name:      pvc.Name,
				Status:    string(pvc.Status.Phase),
			})
		}
	}()

	// --- ResourceQuotas ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		listNS := namespace
		if listNS == "" {
			listNS = metav1.NamespaceAll
		}
		quotas, err := k8s.CoreV1().ResourceQuotas(listNS).List(ctx, metav1.ListOptions{})
		if err != nil {
			return // non-fatal
		}
		mu.Lock()
		defer mu.Unlock()
		for _, q := range quotas.Items {
			used := formatResourceList(q.Status.Used)
			hard := formatResourceList(q.Status.Hard)
			h.ResourceQuotas = append(h.ResourceQuotas, QuotaInfo{
				Namespace: q.Namespace,
				Name:      q.Name,
				Used:      used,
				Hard:      hard,
			})
		}
	}()

	// --- Metrics ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		if mc == nil {
			return
		}
		listNS := namespace
		if listNS == "" {
			listNS = metav1.NamespaceAll
		}
		podMetrics, err := mc.MetricsV1beta1().PodMetricses(listNS).List(ctx, metav1.ListOptions{})
		if err != nil {
			return // metrics server may not be installed
		}

		mu.Lock()
		defer mu.Unlock()
		h.MetricsAvailable = true

		type entry struct {
			ns, name string
			cpu, mem int64
		}
		var all []entry
		for _, pm := range podMetrics.Items {
			var cpu, mem int64
			for _, c := range pm.Containers {
				cpu += c.Usage.Cpu().MilliValue()
				mem += c.Usage.Memory().Value()
			}
			all = append(all, entry{pm.Namespace, pm.Name, cpu, mem})
		}

		// top 5 by CPU
		sorted := make([]entry, len(all))
		copy(sorted, all)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].cpu > sorted[j].cpu })
		for i := 0; i < 5 && i < len(sorted); i++ {
			h.TopPodsCPU = append(h.TopPodsCPU, PodMetric{
				Namespace: sorted[i].ns,
				Name:      sorted[i].name,
				Value:     fmt.Sprintf("%dm", sorted[i].cpu),
			})
		}

		// top 5 by Memory
		sort.Slice(all, func(i, j int) bool { return all[i].mem > all[j].mem })
		for i := 0; i < 5 && i < len(all); i++ {
			h.TopPodsMemory = append(h.TopPodsMemory, PodMetric{
				Namespace: all[i].ns,
				Name:      all[i].name,
				Value:     formatBytes(all[i].mem),
			})
		}

		// node metrics
		nodeMetrics, err := mc.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
		if err != nil {
			return
		}
		nmMap := make(map[string]struct{ cpu, mem string })
		for _, nm := range nodeMetrics.Items {
			nmMap[nm.Name] = struct{ cpu, mem string }{
				cpu: fmt.Sprintf("%dm", nm.Usage.Cpu().MilliValue()),
				mem: formatBytes(nm.Usage.Memory().Value()),
			}
		}
		for i := range h.Nodes {
			if m, ok := nmMap[h.Nodes[i].Name]; ok {
				h.Nodes[i].CPUUsage = m.cpu
				h.Nodes[i].MemUsage = m.mem
			}
		}
	}()

	wg.Wait()

	if len(errs) > 0 {
		return h, fmt.Errorf("collection errors: %s", strings.Join(errs, "; "))
	}
	return h, nil
}

// buildNodeInfo converts a Node API object to NodeInfo.
func buildNodeInfo(n corev1.Node) NodeInfo {
	info := NodeInfo{
		Name:           n.Name,
		KubeletVersion: n.Status.NodeInfo.KubeletVersion,
	}

	// role
	roles := []string{}
	for k := range n.Labels {
		if strings.HasPrefix(k, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(k, "node-role.kubernetes.io/")
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	if len(roles) > 0 {
		sort.Strings(roles)
		info.Role = strings.Join(roles, ",")
	} else {
		info.Role = "worker"
	}

	// status
	info.Status = "Unknown"
	for _, cond := range n.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			switch cond.Status {
			case corev1.ConditionTrue:
				info.Status = "Ready"
				info.Uptime = formatDuration(time.Since(cond.LastTransitionTime.Time))
			case corev1.ConditionFalse:
				info.Status = "NotReady"
			}
			break
		}
	}

	return info
}

// podProblemReason returns the reason a pod is in a bad state, if any.
func podProblemReason(p corev1.Pod) (string, bool) {
	for _, cs := range p.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			r := cs.State.Waiting.Reason
			if r == "CrashLoopBackOff" || r == "ImagePullBackOff" || r == "ErrImagePull" {
				return r, true
			}
		}
	}
	if p.Status.Phase == corev1.PodFailed {
		return string(p.Status.Phase), true
	}
	return "", false
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func formatBytes(b int64) string {
	const (
		MB = 1024 * 1024
		GB = 1024 * MB
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1fGi", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.0fMi", float64(b)/float64(MB))
	default:
		return fmt.Sprintf("%dKi", b/1024)
	}
}

func formatResourceList(rl corev1.ResourceList) string {
	parts := []string{}
	for k, v := range rl {
		parts = append(parts, fmt.Sprintf("%s:%s", k, v.String()))
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}
