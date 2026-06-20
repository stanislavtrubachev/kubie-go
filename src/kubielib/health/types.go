package health

import "time"

// ClusterHealth holds all collected health data for a cluster.
type ClusterHealth struct {
	Context    string
	APIVersion string
	CollectedAt time.Time

	Nodes      []NodeInfo
	NodeSummary NodeSummary

	Pods      []PodInfo
	PodSummary PodSummary

	NamespaceCount int

	Checks []HealthCheck

	MetricsAvailable bool
	TopPodsCPU       []PodMetric
	TopPodsMemory    []PodMetric

	PVCStatus     []PVCInfo
	ResourceQuotas []QuotaInfo
}

// NodeSummary holds aggregated node counts by status.
type NodeSummary struct {
	Total    int
	Ready    int
	NotReady int
	Unknown  int
}

// NodeInfo holds per-node details.
type NodeInfo struct {
	Name           string
	Status         string // Ready / NotReady / Unknown
	Role           string // control-plane / worker
	KubeletVersion string
	CPUUsage       string // empty if metrics unavailable
	MemUsage       string // empty if metrics unavailable
	Uptime         string
}

// PodSummary holds aggregated pod counts by phase.
type PodSummary struct {
	Total          int
	Running        int
	Pending        int
	Failed         int
	Succeeded      int
	Unknown        int
	CrashLoopBackOff int
	ImagePullBackOff int
}

// PodInfo holds per-pod details for problematic pods.
type PodInfo struct {
	Namespace string
	Name      string
	Status    string
	Reason    string
}

// PodMetric holds a pod resource usage snapshot.
type PodMetric struct {
	Namespace string
	Name      string
	Value     string
}

// HealthCheck represents a single health check result.
type HealthCheck struct {
	Name   string
	Status CheckStatus
	Detail string
}

// CheckStatus is the result of a health check.
type CheckStatus int

const (
	StatusOK      CheckStatus = iota
	StatusWarning
	StatusError
)

// PVCInfo holds a PersistentVolumeClaim status.
type PVCInfo struct {
	Namespace string
	Name      string
	Status    string // Bound / Pending / Lost
}

// QuotaInfo holds a ResourceQuota summary.
type QuotaInfo struct {
	Namespace string
	Name      string
	Used      string
	Hard      string
}
