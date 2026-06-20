package health

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// OutputFormat controls how health data is rendered.
type OutputFormat string

const (
	FormatHuman OutputFormat = "human"
	FormatJSON  OutputFormat = "json"
	FormatYAML  OutputFormat = "yaml"
)

// Render writes ClusterHealth to w in the requested format.
func Render(w io.Writer, h *ClusterHealth, format OutputFormat) error {
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(h)
	case FormatYAML:
		return yaml.NewEncoder(w).Encode(h)
	default:
		renderHuman(w, h)
		return nil
	}
}

func renderHuman(w io.Writer, h *ClusterHealth) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Header
	fmt.Fprintf(w, "\n%s%s=== Cluster Health: %s ===%s\n", colorBold, colorCyan, h.Context, colorReset)
	fmt.Fprintf(w, "Collected at: %s\n\n", h.CollectedAt.Format(time.RFC3339))

	// Health Checks
	fmt.Fprintf(w, "%s%s[ Health Checks ]%s\n", colorBold, colorCyan, colorReset)
	for _, c := range h.Checks {
		icon, color := checkStyle(c.Status)
		fmt.Fprintf(w, "  %s %s%-20s%s %s\n", icon, color, c.Name, colorReset, c.Detail)
	}
	fmt.Fprintln(w)

	// Cluster Summary
	fmt.Fprintf(w, "%s%s[ Cluster Summary ]%s\n", colorBold, colorCyan, colorReset)
	fmt.Fprintf(tw, "  API Version\t%s\n", h.APIVersion)
	fmt.Fprintf(tw, "  Namespaces\t%d\n", h.NamespaceCount)
	fmt.Fprintf(tw, "  Nodes\t%s\n", nodeSummaryStr(h.NodeSummary))
	fmt.Fprintf(tw, "  Pods\t%s\n", podSummaryStr(h.PodSummary))
	tw.Flush()
	fmt.Fprintln(w)

	// Nodes table
	fmt.Fprintf(w, "%s%s[ Nodes ]%s\n", colorBold, colorCyan, colorReset)
	fmt.Fprintf(tw, "  NAME\tROLE\tSTATUS\tVERSION\tCPU\tMEMORY\tUPTIME\n")
	fmt.Fprintf(tw, "  ----\t----\t------\t-------\t---\t------\t------\n")
	for _, n := range h.Nodes {
		statusColor := nodeStatusColor(n.Status)
		cpu := dash(n.CPUUsage)
		mem := dash(n.MemUsage)
		uptime := dash(n.Uptime)
		fmt.Fprintf(tw, "  %s\t%s\t%s%s%s\t%s\t%s\t%s\t%s\n",
			n.Name, n.Role, statusColor, n.Status, colorReset,
			n.KubeletVersion, cpu, mem, uptime)
	}
	tw.Flush()
	fmt.Fprintln(w)

	// Pod problems
	if h.PodSummary.CrashLoopBackOff+h.PodSummary.ImagePullBackOff+h.PodSummary.Failed > 0 {
		fmt.Fprintf(w, "%s%s[ Problematic Pods ]%s\n", colorBold, colorCyan, colorReset)
		fmt.Fprintf(tw, "  NAMESPACE\tNAME\tSTATUS\tREASON\n")
		fmt.Fprintf(tw, "  ---------\t----\t------\t------\n")
		for _, p := range h.Pods {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s%s%s\n",
				p.Namespace, p.Name, p.Status, colorRed, p.Reason, colorReset)
		}
		tw.Flush()
		fmt.Fprintln(w)
	}

	// Metrics
	if h.MetricsAvailable {
		if len(h.TopPodsCPU) > 0 {
			fmt.Fprintf(w, "%s%s[ Top Pods by CPU ]%s\n", colorBold, colorCyan, colorReset)
			fmt.Fprintf(tw, "  NAMESPACE\tNAME\tCPU\n")
			for _, m := range h.TopPodsCPU {
				fmt.Fprintf(tw, "  %s\t%s\t%s\n", m.Namespace, m.Name, m.Value)
			}
			tw.Flush()
			fmt.Fprintln(w)
		}
		if len(h.TopPodsMemory) > 0 {
			fmt.Fprintf(w, "%s%s[ Top Pods by Memory ]%s\n", colorBold, colorCyan, colorReset)
			fmt.Fprintf(tw, "  NAMESPACE\tNAME\tMEMORY\n")
			for _, m := range h.TopPodsMemory {
				fmt.Fprintf(tw, "  %s\t%s\t%s\n", m.Namespace, m.Name, m.Value)
			}
			tw.Flush()
			fmt.Fprintln(w)
		}
	} else {
		fmt.Fprintf(w, "  %s⚠️  Metrics Server not available — resource usage skipped%s\n\n", colorYellow, colorReset)
	}

	// PVCs — only show non-Bound
	var badPVCs []PVCInfo
	for _, p := range h.PVCStatus {
		if p.Status != "Bound" {
			badPVCs = append(badPVCs, p)
		}
	}
	if len(badPVCs) > 0 {
		fmt.Fprintf(w, "%s%s[ PVC Issues ]%s\n", colorBold, colorCyan, colorReset)
		fmt.Fprintf(tw, "  NAMESPACE\tNAME\tSTATUS\n")
		for _, p := range badPVCs {
			fmt.Fprintf(tw, "  %s\t%s\t%s%s%s\n", p.Namespace, p.Name, colorYellow, p.Status, colorReset)
		}
		tw.Flush()
		fmt.Fprintln(w)
	}

	// Resource Quotas
	if len(h.ResourceQuotas) > 0 {
		fmt.Fprintf(w, "%s%s[ Resource Quotas ]%s\n", colorBold, colorCyan, colorReset)
		fmt.Fprintf(tw, "  NAMESPACE\tNAME\tUSED\tHARD\n")
		for _, q := range h.ResourceQuotas {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", q.Namespace, q.Name, q.Used, q.Hard)
		}
		tw.Flush()
		fmt.Fprintln(w)
	}
}

func checkStyle(s CheckStatus) (icon, color string) {
	switch s {
	case StatusOK:
		return "✅", colorGreen
	case StatusWarning:
		return "⚠️ ", colorYellow
	default:
		return "❌", colorRed
	}
}

func nodeStatusColor(status string) string {
	switch status {
	case "Ready":
		return colorGreen
	case "NotReady":
		return colorRed
	default:
		return colorYellow
	}
}

func nodeSummaryStr(s NodeSummary) string {
	parts := []string{fmt.Sprintf("total:%d", s.Total)}
	if s.Ready > 0 {
		parts = append(parts, fmt.Sprintf("%sready:%d%s", colorGreen, s.Ready, colorReset))
	}
	if s.NotReady > 0 {
		parts = append(parts, fmt.Sprintf("%snot-ready:%d%s", colorRed, s.NotReady, colorReset))
	}
	if s.Unknown > 0 {
		parts = append(parts, fmt.Sprintf("%sunknown:%d%s", colorYellow, s.Unknown, colorReset))
	}
	return strings.Join(parts, "  ")
}

func podSummaryStr(s PodSummary) string {
	parts := []string{fmt.Sprintf("total:%d", s.Total)}
	if s.Running > 0 {
		parts = append(parts, fmt.Sprintf("%srunning:%d%s", colorGreen, s.Running, colorReset))
	}
	if s.Pending > 0 {
		parts = append(parts, fmt.Sprintf("%spending:%d%s", colorYellow, s.Pending, colorReset))
	}
	if s.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%sfailed:%d%s", colorRed, s.Failed, colorReset))
	}
	if s.CrashLoopBackOff > 0 {
		parts = append(parts, fmt.Sprintf("%scrashloop:%d%s", colorRed, s.CrashLoopBackOff, colorReset))
	}
	if s.ImagePullBackOff > 0 {
		parts = append(parts, fmt.Sprintf("%simagepull:%d%s", colorRed, s.ImagePullBackOff, colorReset))
	}
	return strings.Join(parts, "  ")
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
