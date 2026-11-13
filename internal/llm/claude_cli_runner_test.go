package llm

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClaudeContinuesTheExactSession(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
prompt=$(cat)
case "$*" in
  *--resume*)
    case "$*" in *claude-test-session*) ;; *) exit 3 ;; esac
    test "$prompt" = "second turn" || exit 4
    reply=continued
    ;;
  *)
    test "$prompt" = "first turn" || exit 5
    reply=question
    ;;
esac
printf '%s\n' '{"type":"system","subtype":"init","session_id":"claude-test-session"}'
printf '{"type":"assistant","message":{"content":[{"type":"text","text":"%s"}]}}\n' "$reply"
printf '{"type":"result","subtype":"success","session_id":"claude-test-session","result":"%s","is_error":false}\n' "$reply"
`
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	runner := &ClaudeCLIRunner{}
	for _, turn := range []struct{ prompt, reply string }{{"first turn", "question"}, {"second turn", "continued"}} {
		var streamed []string
		reply, err := runner.Run(dir, turn.prompt, func(text string) { streamed = append(streamed, text) })
		if err != nil || strings.TrimSpace(reply) != turn.reply {
			t.Fatalf("Run = %q, %v", reply, err)
		}
		count := 0
		for _, text := range streamed {
			if text == turn.reply {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("duplicate or missing reply in %q", streamed)
		}
	}
	if _, err := runner.Run(t.TempDir(), "other project", nil); err == nil {
		t.Fatal("conversation accepted another project")
	}
}

func TestClaudeRejectsFailedOrIncompleteStreams(t *testing.T) {
	for _, events := range []string{
		`{"type":"result","subtype":"error_during_execution","is_error":true,"errors":["not logged in"]}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"partial"}]}}`,
		"not json",
	} {
		runner := &ClaudeCLIRunner{}
		if _, err := runner.readReply(strings.NewReader(events), nil); err == nil {
			t.Fatalf("accepted failed stream %q", events)
		}
	}
}

func TestClaudeIgnoresNestedSessionsAndUsesResultFallback(t *testing.T) {
	events := `{"type":"system","subtype":"init","session_id":"root"}
{"type":"assistant","session_id":"child","parent_tool_use_id":"nested","message":{"content":[{"type":"text","text":"ignore"}]}}
{"type":"result","subtype":"success","session_id":"root","result":"done","is_error":false}
`
	runner := &ClaudeCLIRunner{}
	reply, err := runner.readReply(strings.NewReader(events), nil)
	if err != nil || strings.TrimSpace(reply) != "done" || runner.sessionID != "root" {
		t.Fatalf("reply %q, session %q, error %v", reply, runner.sessionID, err)
	}
}

func TestClaudeStreamsBeforeResult(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	messages := make(chan string, 2)
	finished := make(chan error, 1)
	runner := &ClaudeCLIRunner{}
	go func() {
		_, err := runner.readReply(reader, func(text string) { messages <- text })
		finished <- err
	}()
	event := `{"type":"assistant","message":{"content":[{"type":"text","text":"Checking rules."}]}}` + "\n"
	if _, err := io.WriteString(writer, event); err != nil {
		t.Fatal(err)
	}
	select {
	case text := <-messages:
		if text != "Checking rules." {
			t.Fatal(text)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("response was buffered until completion")
	}
	result := `{"type":"result","subtype":"success","result":"Checking rules.","is_error":false}` + "\n"
	if _, err := io.WriteString(writer, result); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("decoder did not finish")
	}
}
