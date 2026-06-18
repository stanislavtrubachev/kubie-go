package cmd

import (
	"runtime"
	"testing"
)

func TestBinaryNameLinuxAmd64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("skipping test on non-Linux/amd64 platform")
	}
	name, ok := getBinaryName()
	if !ok {
		t.Errorf("expected ok=true, got false")
	}
	if name != "kubie-linux-amd64" {
		t.Errorf("expected kubie-linux-amd64, got %s", name)
	}
}

func TestBinaryNameDarwinAmd64(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "amd64" {
		t.Skip("skipping test on non-Darwin/amd64 platform")
	}
	name, ok := getBinaryName()
	if !ok {
		t.Errorf("expected ok=true, got false")
	}
	if name != "kubie-darwin-amd64" {
		t.Errorf("expected kubie-darwin-amd64, got %s", name)
	}
}

func TestBinaryNameDarwinArm64(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("skipping test on non-Darwin/arm64 platform")
	}
	name, ok := getBinaryName()
	if !ok {
		t.Errorf("expected ok=true, got false")
	}
	if name != "kubie-darwin-arm64" {
		t.Errorf("expected kubie-darwin-arm64, got %s", name)
	}
}
