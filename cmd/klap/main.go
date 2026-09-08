package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/leehyowon14/KLAP-cli/internal/bootstrap"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stderr, bootstrap.Run))
}

func run(ctx context.Context, args []string, stderr io.Writer, runner func(context.Context, []string) error) int {
	if err := runner(ctx, args); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
