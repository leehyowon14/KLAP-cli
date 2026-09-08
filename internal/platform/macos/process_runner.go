package macos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type ProcessSpec struct {
	Name  string
	Args  []string
	Input []byte
}

// ProcessError preserves the underlying start, protocol, exit or context error.
// Stderr never enters the JSON/NDJSON stdout stream.
type ProcessError struct {
	Program string
	Stage   string
	Stderr  string
	Err     error
}

func (e *ProcessError) Error() string {
	message := fmt.Sprintf("%s process %s 실패: %v", e.Program, e.Stage, e.Err)
	if e.Stderr != "" {
		message += "\n" + e.Stderr
	}
	return message
}
func (e *ProcessError) Unwrap() error { return e.Err }

type ProcessRunner struct{}

// Start launches a detached desktop helper and reaps it asynchronously. The
// buffered completion channel carries the same errors as Run without requiring
// a receiver; callers interested only in launch success may discard it.
func (r ProcessRunner) Start(ctx context.Context, spec ProcessSpec) (<-chan error, error) {
	process, err := r.startWithStdout(ctx, spec, false)
	if err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() {
		done <- process.wait(nil)
		close(done)
	}()
	return done, nil
}

// Run consumes stdout while the child is alive, then joins it. A decoder error
// cancels the child before waiting, so malformed output cannot leave it blocked.
func (r ProcessRunner) Run(ctx context.Context, spec ProcessSpec, consume func(io.Reader) error) error {
	process, err := r.start(ctx, spec)
	if err != nil {
		return err
	}
	return process.wait(consume)
}

type runningProcess struct {
	parent      context.Context
	cancel      context.CancelFunc
	command     *exec.Cmd
	stdout      io.ReadCloser
	stderr      *bytes.Buffer
	watcherDone chan struct{}
}

func (r ProcessRunner) start(ctx context.Context, spec ProcessSpec) (*runningProcess, error) {
	return r.startWithStdout(ctx, spec, true)
}

func (ProcessRunner) startWithStdout(ctx context.Context, spec ProcessSpec, capture bool) (*runningProcess, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, &ProcessError{Program: spec.Name, Stage: "start", Err: err}
	}
	child, cancel := context.WithCancel(ctx)
	command := exec.CommandContext(child, spec.Name, spec.Args...)
	command.Stdin = bytes.NewReader(spec.Input)
	stderr := &bytes.Buffer{}
	command.Stderr = stderr
	// Bound waiting for inherited stderr descriptors after the direct child exits.
	command.WaitDelay = time.Second
	var stdout io.ReadCloser
	if capture {
		var err error
		stdout, err = command.StdoutPipe()
		if err != nil {
			cancel()
			return nil, &ProcessError{Program: spec.Name, Stage: "start", Err: err}
		}
	}
	// A detached launcher inherits no stdout pipe, so a browser cannot delay Wait.
	if err := command.Start(); err != nil {
		cancel()
		if stdout != nil {
			_ = stdout.Close()
		}
		return nil, &ProcessError{Program: spec.Name, Stage: "start", Err: err}
	}
	process := &runningProcess{parent: ctx, cancel: cancel, command: command, stdout: stdout, stderr: stderr, watcherDone: make(chan struct{})}
	go func() {
		defer close(process.watcherDone)
		<-child.Done()
		// Closing also interrupts a decoder when a grandchild inherited stdout.
		if stdout != nil {
			_ = stdout.Close()
		}
	}()
	return process, nil
}

// wait has exactly one caller. cancel is safe to call concurrently.
func (p *runningProcess) wait(consume func(io.Reader) error) error {
	if consume == nil {
		consume = func(r io.Reader) error { _, err := io.Copy(io.Discard, r); return err }
	}
	var decodeErr error
	if p.stdout != nil {
		decodeErr = consume(p.stdout)
		if decodeErr != nil {
			p.cancel()
		}
		_ = p.stdout.Close()
	}
	waitErr := p.command.Wait()
	p.cancel()
	<-p.watcherDone
	stage, err := "exit", waitErr
	if decodeErr != nil {
		stage, err = "decode", decodeErr
	}
	if ctxErr := p.parent.Err(); ctxErr != nil {
		stage, err = "cancel", ctxErr
	}
	if err == nil {
		return nil
	}
	// Keep both errors where an independent non-zero exit accompanies bad JSON.
	if decodeErr != nil && waitErr != nil && p.parent.Err() == nil {
		err = errors.Join(err, waitErr)
	}
	return &ProcessError{Program: p.command.Path, Stage: stage, Stderr: strings.TrimSpace(p.stderr.String()), Err: err}
}

func bridgeSpec(path string, input []byte) ProcessSpec {
	path = strings.TrimSpace(path)
	if strings.HasSuffix(path, ".swift") {
		return ProcessSpec{Name: "swift", Args: []string{path}, Input: input}
	}
	return ProcessSpec{Name: path, Input: input}
}
