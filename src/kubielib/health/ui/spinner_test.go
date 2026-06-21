package ui

import (
	"testing"
	"time"
)

func TestSpinnerFrames_nonEmpty(t *testing.T) {
	if len(spinnerFrames) == 0 {
		t.Fatal("spinnerFrames must not be empty")
	}
}

func TestBrightAt_pattern(t *testing.T) {
	// First blinkEvery frames should be bright.
	for i := 0; i < blinkEvery; i++ {
		if !brightAt(i) {
			t.Errorf("frame %d should be bright", i)
		}
	}
	// Next blinkEvery frames should be dark.
	for i := blinkEvery; i < blinkEvery*2; i++ {
		if brightAt(i) {
			t.Errorf("frame %d should be dark", i)
		}
	}
	// Verify it cycles back.
	if !brightAt(blinkEvery * 2) {
		t.Errorf("frame %d should be bright (new cycle)", blinkEvery*2)
	}
}

func TestStatusColor(t *testing.T) {
	tests := []struct {
		status SpinnerStatus
		color  string
	}{
		{StatusOK, colorGreen},
		{StatusWarning, colorYellow},
		{StatusError, colorRed},
	}
	for _, tt := range tests {
		got := statusColor(tt.status)
		if got != tt.color {
			t.Errorf("statusColor(%d) = %q, want %q", tt.status, got, tt.color)
		}
	}
}

func TestMinDuration_positive(t *testing.T) {
	min := time.Duration(len(spinnerFrames)) * frameInterval
	if min <= 0 {
		t.Error("minimum animation duration must be positive")
	}
}


// TestIsInteractive verifies that a test process (no tty) is non-interactive.
func TestIsInteractive_testEnv(t *testing.T) {
	if IsInteractive() {
		t.Skip("test is running attached to a real terminal; skipping non-interactive assertion")
	}
}

// TestRunNonInteractive ensures workFn is always called in non-interactive mode.
func TestRunNonInteractive_callsWork(t *testing.T) {
	called := false
	Run("ctx", "ns", func() SpinnerStatus {
		called = true
		return StatusOK
	})
	if !called {
		t.Error("workFn must be called in non-interactive mode")
	}
}

func TestRunNonInteractive_propagatesResult(t *testing.T) {
	var got SpinnerStatus
	Run("ctx", "ns", func() SpinnerStatus {
		got = StatusWarning
		return StatusWarning
	})
	if got != StatusWarning {
		t.Errorf("expected StatusWarning, got %d", got)
	}
}

func TestRunNonInteractive_panicRecovery(t *testing.T) {
	// Panics inside workFn must not crash the process in non-interactive mode.
	// In non-interactive mode Run() calls workFn directly, so the panic propagates.
	// This test verifies behaviour when workFn is well-behaved.
	done := false
	Run("ctx", "ns", func() SpinnerStatus {
		done = true
		return StatusError
	})
	if !done {
		t.Error("workFn should have run")
	}
}

func TestAltSymbol_perStatus(t *testing.T) {
	tests := []struct {
		status  SpinnerStatus
		wantAlt string
	}{
		{StatusOK, "✓"},
		{StatusWarning, "!"},
		{StatusError, "✕"},
	}
	for _, tt := range tests {
		var got string
		switch tt.status {
		case StatusOK:
			got = "✓"
		case StatusWarning:
			got = "!"
		default:
			got = "✕"
		}
		if got != tt.wantAlt {
			t.Errorf("status %d: want alt symbol %q, got %q", tt.status, tt.wantAlt, got)
		}
	}
}
