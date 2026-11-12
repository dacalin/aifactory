package llm

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunContinuesTheSameConversation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable fixtures; stream parsing is tested separately")
	}
	dir := t.TempDir()
	script := `#!/bin/sh
prompt=$(cat)
case "$*" in
  *resume*)
    case "$*" in
      *test-conversation*) ;;
      *) exit 3 ;;
    esac
    test "$prompt" = "second turn" || exit 4
    printf '%s\n' '{"type":"item.completed","item":{"type":"agent_message","text":"continued"}}'
    ;;
  *)
    test "$prompt" = "first turn" || exit 5
    printf '%s\n' '{"type":"thread.started","thread_id":"test-conversation"}'
    printf '%s\n' '{"type":"item.completed","item":{"type":"agent_message","text":"question"}}'
    ;;
esac
printf '%s\n' '{"type":"turn.completed"}'
`
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	runner := &CodexCLIRunner{}
	for _, turn := range []struct{ prompt, reply string }{
		{"first turn", "question"},
		{"second turn", "continued"},
	} {
		reply, err := runner.Run(dir, turn.prompt, nil)
		if err != nil || strings.TrimSpace(reply) != turn.reply {
			t.Fatalf("Run(%q) = %q, %v", turn.prompt, reply, err)
		}
	}
	if _, err := runner.Run(t.TempDir(), "different project", nil); err == nil {
		t.Fatal("conversation accepted a different project")
	}
}

func TestReadReplyStreamsBeforeTheTurnFinishes(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	messages := make(chan string, 1)
	finished := make(chan error, 1)
	runner := &CodexCLIRunner{}
	go func() {
		_, err := runner.readReply(reader, func(message string) { messages <- message })
		finished <- err
	}()
	event := `{"type":"item.completed","item":{"type":"agent_message","text":"Checking existing rules."}}` + "\n"
	if _, err := io.WriteString(writer, event); err != nil {
		t.Fatal(err)
	}
	select {
	case message := <-messages:
		if message != "Checking existing rules." {
			t.Fatalf("unexpected streamed message %q", message)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("message was buffered until the turn finished")
	}
	select {
	case <-finished:
		t.Fatal("reader finished before the stream was closed")
	default:
	}
	writer.Close()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reader did not finish after closing the stream")
	}
}

func TestReadReplyRejectsFailedOrMalformedTurns(t *testing.T) {
	for _, events := range []string{
		`{"type":"turn.failed","error":{"message":"provider unavailable"}}`,
		`{"type":"error","message":"connection failed"}`,
		"invalid json",
	} {
		runner := &CodexCLIRunner{}
		if _, err := runner.readReply(strings.NewReader(events), nil); err == nil {
			t.Fatalf("accepted failed stream %q", events)
		}
	}
}
