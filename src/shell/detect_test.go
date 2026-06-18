package shell

import "testing"

func TestParseCommandSimple(t *testing.T) {
	result := ParseCommand("bash")
	if result != "bash" {
		t.Errorf("ParseCommand(\"bash\") = %q, want %q", result, "bash")
	}
}

func TestParseCommandWithArgs(t *testing.T) {
	result := ParseCommand("bash --rcfile hello.sh")
	if result != "bash" {
		t.Errorf("ParseCommand(\"bash --rcfile hello.sh\") = %q, want %q", result, "bash")
	}
}

func TestParseCommandWithPath(t *testing.T) {
	result := ParseCommand("/bin/bash")
	if result != "bash" {
		t.Errorf("ParseCommand(\"/bin/bash\") = %q, want %q", result, "bash")
	}
}

func TestParseCommandWithPathAndArgs(t *testing.T) {
	result := ParseCommand("/bin/bash --rcfile hello.sh")
	if result != "bash" {
		t.Errorf("ParseCommand(\"/bin/bash --rcfile hello.sh\") = %q, want %q", result, "bash")
	}
}

func TestParseCommandLoginShell(t *testing.T) {
	result := ParseCommand("-zsh")
	if result != "zsh" {
		t.Errorf("ParseCommand(\"-zsh\") = %q, want %q", result, "zsh")
	}
}

func TestParseCommandVersionedInterpreter(t *testing.T) {
	result := ParseCommand("python3.8")
	if result != "python" {
		t.Errorf("ParseCommand(\"python3.8\") = %q, want %q", result, "python")
	}
}

func TestParseCommandNu(t *testing.T) {
	result := ParseCommand("/bin/nu")
	if result != "nu" {
		t.Errorf("ParseCommand(\"/bin/nu\") = %q, want %q", result, "nu")
	}
}
