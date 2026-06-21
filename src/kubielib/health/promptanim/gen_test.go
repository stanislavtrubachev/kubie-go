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
		"_kubie_build_prefix",
		"__kubie_orig_ps1__",
		"__kubie_applied_prefix__",
		"zle -F",
		"zle reset-prompt",
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

func TestZshCode_prependsNotReplaces(t *testing.T) {
	code := ZshCode("ctx", "ns", "/tmp/f")
	// Must save the original PS1 and prepend, not assign a fixed PS1 wholesale
	if !strings.Contains(code, "__kubie_orig_ps1__") {
		t.Error("ZshCode must save original PS1 into __kubie_orig_ps1__")
	}
	if !strings.Contains(code, `PS1="${__kubie_prefix__}${__kubie_orig_ps1__}"`) {
		t.Error("ZshCode must build PS1 as prefix+orig, not a fixed replacement")
	}
}

func TestZshCode_staticPsGuard(t *testing.T) {
	code := ZshCode("ctx", "ns", "/tmp/f")
	// Guard that prevents double-prepend: precmd compares PS1 to (applied_prefix+orig)
	// to detect whether the theme changed PS1 or we did.
	if !strings.Contains(code, "__kubie_applied_prefix__") {
		t.Error("ZshCode must track applied_prefix to detect theme-vs-kubie PS1 changes")
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
	// Real-time updates use zle -F fd watcher, not SIGWINCH
	if strings.Contains(code, "SIGWINCH") || strings.Contains(code, "TRAPWINCH") {
		t.Error("ZshCode must not use SIGWINCH/TRAPWINCH; use zle -F instead")
	}
}

func TestZshCode_fileFormatInDaemon(t *testing.T) {
	code := ZshCode("ctx", "ns", "/tmp/f")
	// Daemon writes "char:bright", prefix builder reads it via *:0/*:1 patterns
	if !strings.Contains(code, "*:1") || !strings.Contains(code, "*:0") {
		t.Error("ZshCode prefix builder must handle 'char:bright' file format")
	}
}
