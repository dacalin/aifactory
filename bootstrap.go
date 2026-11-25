package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/dacalin/aifactory/internal/agents"
	"github.com/dacalin/aifactory/internal/agents/llm"
	"github.com/dacalin/aifactory/internal/agents/llm/claude"
	"github.com/dacalin/aifactory/internal/agents/llm/codex"
	"github.com/dacalin/aifactory/internal/app/project"
	"github.com/dacalin/aifactory/internal/app/task"
	"github.com/dacalin/aifactory/internal/commands"
	"github.com/dacalin/aifactory/internal/config"
	"github.com/dacalin/aifactory/internal/server"
	"github.com/dacalin/aifactory/internal/server/dialog"
	"github.com/dacalin/aifactory/internal/server/instance"
	"github.com/dacalin/aifactory/internal/storage/sqlite"
	workspacegit "github.com/dacalin/aifactory/internal/workspace/git"
	"github.com/dacalin/aifactory/web"

	"github.com/spf13/cobra"
)

func bootstrap() *cobra.Command {
	return commands.NewRootCommand(newRuleAgent, runServer)
}

func newRuleAgent() (agents.Agent, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(projectDir)
	if err != nil {
		return nil, err
	}
	var runner agents.AgentRunner
	writable := llm.WithWritableDirs(cfg.RulesDir())
	switch cfg.RuleAgent.Agent {
	case "codex":
		runner = codex.NewRunner(cfg.RuleAgent.Model, cfg.RuleAgent.Effort, writable)
	case "claude":
		runner = claude.NewRunner(cfg.RuleAgent.Model, cfg.RuleAgent.Effort, writable)
	default:
		return nil, fmt.Errorf("unsupported rule-agent.agent %q", cfg.RuleAgent.Agent)
	}
	return agents.NewRuleAgent(runner, cfg)
}

func runServer(ctx context.Context, output io.Writer, noBrowser bool) error {
	settings, err := config.LoadGlobal()
	if err != nil {
		return err
	}
	serverLock, err := instance.Acquire(filepath.Join(settings.DataDir, "server.lock"))
	if err != nil {
		return err
	}
	defer serverLock.Close()
	database, err := sqlite.Open(settings.DatabasePath)
	if err != nil {
		return err
	}
	defer database.Close()

	projects := project.NewService(database.Projects(), workspacegit.Repository{})
	tasks := task.NewService(projects, database.Tasks())
	handler := server.NewHandler(projects, tasks, dialog.NativeFolderPicker{}, web.Files())
	listener, err := net.Listen("tcp4", settings.ListenAddress)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", settings.ListenAddress, err)
	}
	address := listener.Addr().String()
	url := "http://" + address
	httpServer := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- httpServer.Serve(listener) }()

	fmt.Fprintf(output, "AI Factory is running at %s\n", url)
	if !noBrowser {
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(output, "Could not open a browser automatically: %v\n", err)
		}
	}
	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownContext); err != nil {
			_ = httpServer.Close()
			return fmt.Errorf("shut down local server: %w", err)
		}
		return nil
	case err := <-serveErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve local UI: %w", err)
	}
}

func openBrowser(url string) error {
	var command string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
	case "linux":
		command = "xdg-open"
	default:
		return fmt.Errorf("unsupported desktop platform %q", runtime.GOOS)
	}
	process := exec.Command(command, url)
	if err := process.Start(); err != nil {
		return err
	}
	go func() { _ = process.Wait() }()
	return nil
}
