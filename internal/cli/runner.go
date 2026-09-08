package cli

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"io"
	"time"
)

type Runner struct {
	Service *app.Service
	Out     io.Writer
	ErrOut  io.Writer
	Opener  func(string, string) error
	Clock   func() time.Time
}

func (r Runner) normalized() Runner {
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
