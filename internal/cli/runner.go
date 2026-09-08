package cli

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"io"
	"time"
)

// TerminalCapabilities is detected by bootstrap and stays deterministic in renderers.
type TerminalCapabilities struct {
	Hyperlinks      bool
	InPlaceProgress bool
}

type Runner struct {
	In         io.Reader
	AuthPrompt func(context.Context, io.Reader, io.Writer) (AuthCredentials, error)
	Terminal   TerminalCapabilities
	Service    *app.Service
	Out        io.Writer
	ErrOut     io.Writer
	Opener     func(string, string) error
	Clock      func() time.Time
}

func (r Runner) normalized() Runner {
	if r.AuthPrompt == nil {
		r.AuthPrompt = runAuthForm
	}
	if r.Out == nil {
		r.Out = io.Discard
	}
	if r.ErrOut == nil {
		r.ErrOut = io.Discard
	}
	if r.Clock == nil {
		r.Clock = time.Now
	}
	if r.Opener == nil {
		r.Opener = func(string, string) error { return errors.New("external opener is not configured") }
	}
	return r
}

func (r Runner) Run(ctx context.Context, args []string) error {
	r = r.normalized()
	if len(args) == 0 {
		return errors.New("missing CLI command")
	}
	return dispatchCommand(ctx, r, args, commandRegistry())
}
