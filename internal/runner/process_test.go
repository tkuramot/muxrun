package runner

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestWaitForExit_ProcessGone(t *testing.T) {
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("running helper: %v", err)
	}
	if !waitForExit(cmd.Process.Pid, time.Second) {
		t.Error("expected an exited process to be reported as gone")
	}
}

func TestWaitForExit_ProcessAlive(t *testing.T) {
	start := time.Now()
	if waitForExit(os.Getpid(), 100*time.Millisecond) {
		t.Error("expected a live process to still be running")
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Errorf("returned after %v, expected to wait for the full timeout", elapsed)
	}
}

func TestTerminateProcess(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting helper: %v", err)
	}
	pid := cmd.Process.Pid
	// Reap the helper the way its real parent would; an unreaped child would
	// linger as a zombie and still answer signal 0.
	reaped := make(chan struct{})
	go func() {
		cmd.Wait()
		close(reaped)
	}()
	defer func() { <-reaped }()

	start := time.Now()
	if err := TerminateProcess(pid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > killTimeout {
		t.Errorf("took %v, expected SIGTERM to be enough", elapsed)
	}
}

func TestTerminateProcess_NoSuchProcess(t *testing.T) {
	if err := TerminateProcess(-1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
