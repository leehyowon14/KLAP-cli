package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/kw-klap/klap-cli/internal/cli"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stderr, cli.Run))
}

func run(ctx context.Context, args []string, stderr io.Writer, runner func(context.Context, []string) error) int {
	if err := runner(ctx, args); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
