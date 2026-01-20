//go:build integration

package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(runIntegrationTests(m))
}

func runIntegrationTests(m *testing.M) int {
	sanitizeIntegrationEnv()
	return m.Run()
}

func sanitizeIntegrationEnv() {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GT_") || strings.HasPrefix(env, "BD_") || strings.HasPrefix(env, "BEADS_") {
			if key := strings.SplitN(env, "=", 2)[0]; key != "" {
				os.Unsetenv(key)
			}
		}
	}
	os.Setenv("BEADS_NO_DAEMON", "1")
}
