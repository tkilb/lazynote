package digest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// maxOutputBytes caps what is read from the command: the answer plus the
// CLI's own envelope.
const maxOutputBytes = 4 << 20

// killGrace is how long a stopped command may keep its pipes open.
const killGrace = time.Second

var (
	// ErrBackendMissing reports a command that cannot be found or run.
	ErrBackendMissing = errors.New("the agent command was not found")
	// ErrBackendFailed reports a command that exited with an error or said
	// it failed.
	ErrBackendFailed = errors.New("the agent command failed")
	// ErrTimeout reports a command stopped for taking too long.
	ErrTimeout = errors.New("the agent took too long")
	// ErrNoOutput reports a command that finished without an answer.
	ErrNoOutput = errors.New("the agent returned no answer")
	// ErrBadSettings reports a missing or unsafe command, model, prompt or
	// timeout.
	ErrBadSettings = errors.New("the agent settings are not valid")
	// ErrEmptySource reports a source with no text.
	ErrEmptySource = errors.New("there is nothing to import")
)

// CLI is a Digester that runs the Claude Code CLI in print mode: no tools,
// no saved session, no MCP servers, in an empty temporary directory, with the
// prompt as the whole system prompt and the source on standard input. The
// source is never a command-line argument. It adds no API key handling: the
// command uses its own login.
type CLI struct {
	// Command is the program to run, "claude" by default in the config.
	Command string
	// Model is the model name or alias.
	Model   string
	Timeout time.Duration
	// Prompt is the full prompt text, from ReadPrompt.
	Prompt string
	// Sources are the configured sources, for Parse.
	Sources []string
}

// cliEnvelope is the JSON the CLI prints with --output-format json. The answer
// checked against the schema is structured_output; result holds it as text.
type cliEnvelope struct {
	IsError          bool            `json:"is_error"`
	Result           string          `json:"result"`
	StructuredOutput json.RawMessage `json:"structured_output"`
}

// Digest runs the agent over source and returns its checked answer. Errors
// never contain the source, the command's output or its error stream.
func (c CLI) Digest(ctx context.Context, source string) (Digest, error) {
	if err := c.validate(); err != nil {
		return Digest{}, err
	}
	if strings.TrimSpace(source) == "" {
		return Digest{}, ErrEmptySource
	}
	out, err := c.run(ctx, source)
	if err != nil {
		return Digest{}, err
	}
	return c.answer(out)
}

func (c CLI) validate() error {
	if strings.TrimSpace(c.Command) == "" || strings.TrimSpace(c.Prompt) == "" ||
		c.Model == "" || strings.HasPrefix(c.Model, "-") || c.Timeout <= 0 {
		return ErrBadSettings
	}
	return nil
}

func (c CLI) args() []string {
	return []string{
		"-p",
		"--model", c.Model,
		"--tools", "",
		"--disable-slash-commands",
		"--strict-mcp-config",
		"--no-session-persistence",
		"--output-format", "json",
		"--system-prompt", c.Prompt,
		"--json-schema", Schema,
	}
}

func (c CLI) run(ctx context.Context, source string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "lazynote-digest-")
	if err != nil {
		return nil, fmt.Errorf("%w: no temporary directory", ErrBackendFailed)
	}
	defer os.RemoveAll(dir)

	runCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, c.Command, c.args()...)
	cmd.Dir = dir
	cmd.WaitDelay = killGrace
	cmd.Stdin = strings.NewReader(source)
	var stdout limitedBuffer
	cmd.Stdout = &stdout
	// The error stream is discarded: it may echo the source.

	runErr := cmd.Run()
	switch {
	case ctx.Err() != nil:
		return nil, ctx.Err()
	case errors.Is(runCtx.Err(), context.DeadlineExceeded):
		return nil, ErrTimeout
	case stdout.over:
		return nil, ErrTooLarge
	case runErr == nil:
		return stdout.buf.Bytes(), nil
	}
	var exit *exec.ExitError
	if errors.As(runErr, &exit) {
		return nil, fmt.Errorf("%w (exit %d)", ErrBackendFailed, exit.ExitCode())
	}
	return nil, ErrBackendMissing
}

func (c CLI) answer(out []byte) (Digest, error) {
	var env cliEnvelope
	if err := json.Unmarshal(out, &env); err != nil {
		return Digest{}, ErrBadJSON
	}
	if env.IsError {
		return Digest{}, ErrBackendFailed
	}
	answer := bytes.TrimSpace(env.StructuredOutput)
	if len(answer) == 0 || string(answer) == "null" {
		answer = bytes.TrimSpace([]byte(env.Result))
	}
	if len(answer) == 0 {
		return Digest{}, ErrNoOutput
	}
	return Parse(answer, c.Sources)
}

// limitedBuffer collects up to maxOutputBytes and drops the rest, noting
// that it did, so a runaway command cannot fill memory.
type limitedBuffer struct {
	buf  bytes.Buffer
	over bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	room := maxOutputBytes - b.buf.Len()
	if len(p) > room {
		b.over = true
		p = p[:max(room, 0)]
	}
	b.buf.Write(p)
	return len(p), nil
}

var _ io.Writer = (*limitedBuffer)(nil)
