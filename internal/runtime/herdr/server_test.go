package herdr

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

// TestServerRunningRejectsStaleSocket verifies that filesystem residue cannot
// suppress startup of a stopped named Herdr session.
func TestServerRunningRejectsStaleSocket(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "herdr.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen on test socket: %v", err)
	}
	listener.(*net.UnixListener).SetUnlinkOnClose(false)
	if err := listener.Close(); err != nil {
		t.Fatalf("close test socket: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(socketPath) })

	status := filepath.Join(dir, "herdr")
	if err := os.WriteFile(status, []byte("#!/bin/sh\nprintf '%s\\n' '{\"running\":false}'\n"), 0o755); err != nil {
		t.Fatalf("write fake herdr: %v", err)
	}
	c := newClient("stopped", dir)
	c.bin = status
	c.sockPath = socketPath
	if c.serverRunning() {
		t.Fatal("serverRunning() = true for stale socket")
	}
}

// TestServerRunningUsesHerdrStatus verifies that authoritative status wins
// even when a custom socket path is absent.
func TestServerRunningUsesHerdrStatus(t *testing.T) {
	dir := t.TempDir()
	status := filepath.Join(dir, "herdr")
	if err := os.WriteFile(status, []byte("#!/bin/sh\nprintf '%s\\n' '{\"running\":true}'\n"), 0o755); err != nil {
		t.Fatalf("write fake herdr: %v", err)
	}
	c := newClient("running", dir)
	c.bin = status
	c.sockPath = filepath.Join(dir, "missing.sock")
	if !c.serverRunning() {
		t.Fatal("serverRunning() = false for running Herdr status")
	}
}
