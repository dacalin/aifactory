package llm_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dacalin/aifactory/internal/agents"
	"github.com/dacalin/aifactory/internal/agents/llm/claude"
	"github.com/dacalin/aifactory/internal/agents/llm/codex"
)

func TestProviderRunnerCancelsActiveCLI(t *testing.T) {
	providers := []struct {
		name, event string
		newRunner   func() agents.AgentRunner
	}{
		{"codex", `{"type":"turn.started"}`, func() agents.AgentRunner { return codex.NewRunner("", "") }},
		{"claude", `{"type":"system","subtype":"init"}`, func() agents.AgentRunner { return claude.NewRunner("", "") }},
	}
	for _, provider := range providers {
		t.Run(provider.name, func(t *testing.T) {
			dir := t.TempDir()
			script := "#!/bin/sh\nprintf '%s\\n' \"$$\" > provider.pid\nsleep 30 &\nprintf '%s\\n' '" + provider.event + "'\nwait\n"
			if err := os.WriteFile(filepath.Join(dir, provider.name), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			started, finished := make(chan struct{}, 1), make(chan error, 1)
			go func() {
				_, err := provider.newRunner().Run(ctx, dir, "cancel this turn", func(string) {
					started <- struct{}{}
				})
				finished <- err
			}()
			select {
			case <-started:
				cancel()
			case err := <-finished:
				t.Fatalf("runner exited before cancellation: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("provider did not start")
			}
			select {
			case err := <-finished:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("Run error = %v, want context.Canceled", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("runner did not return after cancellation")
			}
			assertProviderExited(t, dir)
		})
	}
}

func assertProviderExited(t *testing.T, dir string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "provider.pid"))
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		t.Fatalf("invalid provider PID %q: %v", data, err)
	}
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("provider process %d was not reaped: %v", pid, err)
	}
}
