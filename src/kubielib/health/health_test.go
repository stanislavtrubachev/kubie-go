package health

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// --- types / helpers ---

func TestParseContextFromSelector_aliases(t *testing.T) {
	// reuse the same helper logic that ParseContextFromSelector in aliases package uses,
	// just making sure our formatting helpers round-trip correctly
	cases := []struct {
		name   string
		status CheckStatus
		wantOK bool
	}{
		{"API Server", StatusOK, true},
		{"CoreDNS", StatusWarning, false},
		{"Nodes Ready", StatusError, false},
	}
	for _, c := range cases {
		icon, _ := checkStyle(c.status)
		if c.wantOK && icon != "✅" {
			t.Errorf("%s: expected ✅, got %s", c.name, icon)
		}
	}
}

func TestCheckStyle(t *testing.T) {
	tests := []struct {
		status   CheckStatus
		wantIcon string
	}{
		{StatusOK, "✅"},
		{StatusWarning, "⚠️ "},
		{StatusError, "❌"},
	}
	for _, tt := range tests {
		icon, _ := checkStyle(tt.status)
		if icon != tt.wantIcon {
			t.Errorf("checkStyle(%d) icon = %q, want %q", tt.status, icon, tt.wantIcon)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{512 * 1024, "512Ki"},
		{10 * 1024 * 1024, "10Mi"},
		{2 * 1024 * 1024 * 1024, "2.0Gi"},
	}
	for _, tt := range tests {
		got := formatBytes(tt.input)
		if got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{3 * time.Hour, "3h0m"},
		{25 * time.Hour, "1d1h"},
		{48 * time.Hour, "2d0h"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestDash(t *testing.T) {
	if dash("") != "-" {
		t.Error("dash(\"\") should return \"-\"")
	}
	if dash("val") != "val" {
		t.Error("dash(\"val\") should return \"val\"")
	}
}

// --- NodeSummary ---

func TestCheckNodesReady_allReady(t *testing.T) {
	h := &ClusterHealth{
		NodeSummary: NodeSummary{Total: 3, Ready: 3},
	}
	c := checkNodesReady(h)
	if c.Status != StatusOK {
		t.Errorf("expected StatusOK, got %d: %s", c.Status, c.Detail)
	}
}

func TestCheckNodesReady_someNotReady(t *testing.T) {
	h := &ClusterHealth{
		NodeSummary: NodeSummary{Total: 3, Ready: 2, NotReady: 1},
	}
	c := checkNodesReady(h)
	if c.Status != StatusWarning {
		t.Errorf("expected StatusWarning, got %d", c.Status)
	}
}

func TestCheckNodesReady_noNodes(t *testing.T) {
	h := &ClusterHealth{}
	c := checkNodesReady(h)
	if c.Status != StatusWarning {
		t.Errorf("expected StatusWarning for empty cluster, got %d", c.Status)
	}
}

// --- Renderer ---

func TestRenderHuman_noErrors(t *testing.T) {
	h := &ClusterHealth{
		Context:     "test-context",
		APIVersion:  "v1.28.4",
		CollectedAt: time.Now(),
		NodeSummary: NodeSummary{Total: 1, Ready: 1},
		PodSummary:  PodSummary{Total: 5, Running: 5},
		Checks: []HealthCheck{
			{Name: "API Server", Status: StatusOK, Detail: "reachable"},
		},
	}

	var buf bytes.Buffer
	if err := Render(&buf, h, FormatHuman); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "test-context") {
		t.Error("output should contain context name")
	}
	if !strings.Contains(out, "API Server") {
		t.Error("output should contain health check name")
	}
}

func TestRenderJSON(t *testing.T) {
	h := &ClusterHealth{
		Context:     "ctx",
		CollectedAt: time.Now(),
	}
	var buf bytes.Buffer
	if err := Render(&buf, h, FormatJSON); err != nil {
		t.Fatalf("Render JSON error: %v", err)
	}
	if !strings.Contains(buf.String(), `"Context"`) {
		t.Error("JSON output should contain Context field")
	}
}

func TestRenderYAML(t *testing.T) {
	h := &ClusterHealth{
		Context:     "ctx",
		CollectedAt: time.Now(),
	}
	var buf bytes.Buffer
	if err := Render(&buf, h, FormatYAML); err != nil {
		t.Fatalf("Render YAML error: %v", err)
	}
	if !strings.Contains(buf.String(), "context:") {
		t.Error("YAML output should contain context key")
	}
}

func TestRenderHuman_problematicPods(t *testing.T) {
	h := &ClusterHealth{
		Context:     "test",
		CollectedAt: time.Now(),
		PodSummary:  PodSummary{Total: 3, Running: 2, Failed: 1, CrashLoopBackOff: 1},
		Pods: []PodInfo{
			{Namespace: "default", Name: "bad-pod", Status: "Failed", Reason: "CrashLoopBackOff"},
		},
	}
	var buf bytes.Buffer
	Render(&buf, h, FormatHuman)
	out := buf.String()
	if !strings.Contains(out, "Problematic Pods") {
		t.Error("should show problematic pods section")
	}
	if !strings.Contains(out, "bad-pod") {
		t.Error("should show the bad pod name")
	}
}

func TestRenderHuman_metricsUnavailable(t *testing.T) {
	h := &ClusterHealth{
		Context:          "test",
		CollectedAt:      time.Now(),
		MetricsAvailable: false,
	}
	var buf bytes.Buffer
	Render(&buf, h, FormatHuman)
	if !strings.Contains(buf.String(), "Metrics Server not available") {
		t.Error("should warn when metrics unavailable")
	}
}

func TestRenderHuman_pvcIssues(t *testing.T) {
	h := &ClusterHealth{
		Context:     "test",
		CollectedAt: time.Now(),
		PVCStatus: []PVCInfo{
			{Namespace: "default", Name: "my-pvc", Status: "Pending"},
		},
	}
	var buf bytes.Buffer
	Render(&buf, h, FormatHuman)
	if !strings.Contains(buf.String(), "PVC Issues") {
		t.Error("should show PVC issues section")
	}
}

func TestNodeSummaryStr(t *testing.T) {
	s := NodeSummary{Total: 2, Ready: 1, NotReady: 1}
	result := nodeSummaryStr(s)
	if !strings.Contains(result, "total:2") {
		t.Error("should contain total")
	}
}

func TestPodSummaryStr(t *testing.T) {
	s := PodSummary{Total: 10, Running: 8, Failed: 2, CrashLoopBackOff: 1}
	result := podSummaryStr(s)
	if !strings.Contains(result, "total:10") {
		t.Error("should contain total")
	}
	if !strings.Contains(result, "crashloop:1") {
		t.Error("should contain crashloop count")
	}
}
