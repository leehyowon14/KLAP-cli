package bootstrap

import (
	"context"
	"os"
	"time"

	"github.com/leehyowon14/KLAP-cli/internal/app"
	"github.com/leehyowon14/KLAP-cli/internal/cli"
	"github.com/leehyowon14/KLAP-cli/internal/platform/macos"
	"github.com/leehyowon14/KLAP-cli/internal/tui"
)

func Run(ctx context.Context, args []string) error {
	return runWithFactory(ctx, args, NewService, func(ctx context.Context, args []string, s *app.Service) error {
		return (cli.Runner{Service: s, In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr, Opener: func(target, kind string) error { return macos.OpenExternal(ctx, target, kind) }, Clock: time.Now, Terminal: detectTerminalCapabilities(os.Stdout, os.Getenv)}).Run(ctx, args)
	}, func(ctx context.Context, s *app.Service) error { return tui.Run(ctx, s, macos.OpenExternal) })
}

func runWithFactory(ctx context.Context, args []string, create func() (*app.Service, error), runCLI func(context.Context, []string, *app.Service) error, runTUI func(context.Context, *app.Service) error) error {
	service, err := create()
	if err != nil {
		return err
	}
	if len(args) == 0 || args[0] == "tui" {
		return runTUI(ctx, service)
	}
	return runCLI(ctx, args, service)
}
