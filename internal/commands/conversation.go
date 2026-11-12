package commands

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/dacalin/aifactory/internal/agents"
)

func runConversation(agent agents.Agent, prompt string, input io.Reader, output io.Writer) error {
	onMessage := func(message string) { fmt.Fprintln(output, message) }
	_, err := agent.Converse(prompt, onMessage)
	if err != nil {
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
		_, err = agent.Converse(prompt, onMessage)
		if err != nil {
			return err
		}
	}
}
