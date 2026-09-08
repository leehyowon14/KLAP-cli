package bootstrap

import (
	"context"
	"os"
	"time"

	"github.com/leehyowon14/KLAP-cli/internal/app"
	"github.com/leehyowon14/KLAP-cli/internal/cli"
	"github.com/leehyowon14/KLAP-cli/internal/tui"
)

func Run(ctx context.Context, args []string) error {
	return runWithFactory(ctx, args, NewService, func(ctx context.Context, args []string, s *app.Service) error {
		return (cli.Runner{Service: s, Out: os.Stdout, ErrOut: os.Stderr, Opener: openExternal, Clock: time.Now}).Run(ctx, args)
	}, tui.Run)
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
