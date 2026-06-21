package promptanim

import (
	"strings"
	"testing"
)

func TestZshCode_containsKeyParts(t *testing.T) {
	code := ZshCode("k3s", "absolem", "/tmp/kubie-anim-test.txt")
	checks := []string{
		"/tmp/kubie-anim-test.txt",
		"k3s",
		"absolem",
		"__kubie_daemon__",
		"__kubie_precmd__",
		"add-zsh-hook precmd",
		"trap",
		"EXIT",
		"▮",
		"sleep 0.15",
	}
	for _, want := range checks {
		if !strings.Contains(code, want) {
			t.Errorf("ZshCode missing %q", want)
		}
	}
}

func TestBashCode_containsKeyParts(t *testing.T) {
	code := BashCode("k3s", "absolem", "/tmp/kubie-anim-test.txt")
	checks := []string{
		"/tmp/kubie-anim-test.txt",
		"k3s",
		"absolem",
		"__kubie_daemon__",
		"__kubie_build_ps1__",
		"PROMPT_COMMAND",
		"trap",
		"EXIT",
		"▮",
		"sleep 0.15",
	}
	for _, want := range checks {
		if !strings.Contains(code, want) {
			t.Errorf("BashCode missing %q", want)
		}
	}
}

func TestZshCode_emptyNs(t *testing.T) {
	code := ZshCode("prod", "", "/tmp/kubie-anim-test.txt")
	if !strings.Contains(code, "prod") {
		t.Error("ZshCode should contain ctx name")
	}
}

func TestBashCode_emptyNs(t *testing.T) {
	code := BashCode("prod", "", "/tmp/kubie-anim-test.txt")
	if !strings.Contains(code, "prod") {
		t.Error("BashCode should contain ctx name")
	}
}

func TestShellQuote_apostrophe(t *testing.T) {
	got := shellQuote("it's")
	want := `'it'\''s'`
	if got != want {
		t.Errorf("shellQuote(%q) = %q, want %q", "it's", got, want)
	}
}

func TestShellQuote_plain(t *testing.T) {
	got := shellQuote("k3s-prod")
	if got != "'k3s-prod'" {
		t.Errorf("shellQuote plain: got %q", got)
	}
}

func TestZshCode_spinFramesPresent(t *testing.T) {
	code := ZshCode("ctx", "ns", "/tmp/f")
	for _, f := range []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"} {
		if !strings.Contains(code, f) {
			t.Errorf("ZshCode missing Braille frame %q", f)
		}
	}
}

func TestZshCode_noSigwinch(t *testing.T) {
	code := ZshCode("ctx", "ns", "/tmp/f")
	if strings.Contains(code, "SIGWINCH") || strings.Contains(code, "TRAPWINCH") {
		t.Error("ZshCode must not use SIGWINCH/TRAPWINCH (prompt updates on Enter only)")
	}
}

func TestZshCode_fileFormatInDaemon(t *testing.T) {
	code := ZshCode("ctx", "ns", "/tmp/f")
	// Daemon writes "char:bright", hook reads it via *:0/*:1 patterns
	if !strings.Contains(code, "*:1") || !strings.Contains(code, "*:0") {
		t.Error("ZshCode hook must handle 'char:bright' file format")
	}
}
