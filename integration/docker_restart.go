//go:build integration
// +build integration

package integration

import (
	"context"
	"os/exec"
	"testing"
)

func restartOrderContainer(t *testing.T, ctx context.Context) {
	t.Helper()

	cmd := exec.CommandContext(ctx, "docker", "compose", "restart", "order")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docker compose restart order failed: %v\n%s", err, string(out))
	}
}
