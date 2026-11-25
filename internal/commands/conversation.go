package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/dacalin/aifactory/internal/agents"
)

func runConversation(ctx context.Context, agent agents.Agent, prompt string, input io.Reader, output io.Writer) (err error) {
	if closer, ok := agent.(io.Closer); ok {
		defer func() {
			if cleanupErr := closer.Close(); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("clean up rule session: %w", cleanupErr))
			}
		}()
	}
	onMessage := func(message string) { fmt.Fprintln(output, message) }
	if err := converseTurn(ctx, agent, prompt, onMessage); err != nil {
		return err
	}
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for {
		fmt.Fprint(output, "\nYou (/exit to finish)> ")
		if !scanner.Scan() {
			return scanner.Err()
		}
		prompt = strings.TrimSpace(scanner.Text())
		if prompt == "/exit" {
			return nil
		}
		if prompt == "" {
			continue
		}
		if err := converseTurn(ctx, agent, prompt, onMessage); err != nil {
			return err
		}
	}
}

func converseTurn(ctx context.Context, agent agents.Agent, prompt string, onMessage func(string)) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	_, err := agent.ConverseContext(ctx, prompt, onMessage)
	return err
}
