package beads

import (
	"os/exec"
	"testing"
)

func TestCheckBdDaemonHealth_NoDaemonEnv(t *testing.T) {
	t.Setenv("BEADS_NO_DAEMON", "1")
	health, err := CheckBdDaemonHealth()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if health != nil {
		t.Fatalf("expected nil health when BEADS_NO_DAEMON is set, got %+v", health)
	}
}

func TestEnsureBdDaemonHealth_NoDaemonEnv(t *testing.T) {
	t.Setenv("BEADS_NO_DAEMON", "1")
	if warning := EnsureBdDaemonHealth(t.TempDir()); warning != "" {
		t.Fatalf("expected no warning when BEADS_NO_DAEMON is set, got %q", warning)
	}
}

func TestCountBdActivityProcesses(t *testing.T) {
	count := CountBdActivityProcesses()
	if count < 0 {
		t.Errorf("count should be non-negative, got %d", count)
	}
}

func TestCountBdDaemons(t *testing.T) {
	if _, err := exec.LookPath("bd"); err != nil {
		t.Skip("bd not installed")
	}
	count := CountBdDaemons()
	if count < 0 {
		t.Errorf("count should be non-negative, got %d", count)
	}
}

func TestStopAllBdProcesses_DryRun(t *testing.T) {
	daemonsKilled, activityKilled, err := StopAllBdProcesses(true, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if daemonsKilled < 0 || activityKilled < 0 {
		t.Errorf("counts should be non-negative: daemons=%d, activity=%d", daemonsKilled, activityKilled)
	}
}
