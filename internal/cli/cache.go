package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func (r Runner) runCache(ctx context.Context, service *app.Service, args []string) error {
	_ = ctx
	if len(args) == 0 {
		return errors.New("usage: klap cache <status|clear>")
	}
	switch args[0] {
	case "status":
		result, err := service.CacheStatus()
		if err != nil {
			return err
		}
		r.printCacheStatus(result)
		return nil
	case "clear":
		scope := ""
		if len(args) > 1 {
			scope = args[1]
		}
		result, err := service.ClearCacheScope(scope)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(r.Out, "캐시 삭제 완료: %d개\n", result.Removed)
		return nil
	default:
		return fmt.Errorf("unknown cache command: %s", args[0])
	}
}

func (r Runner) printCacheStatus(result app.CacheStatusResult) {
	_, _ = fmt.Fprintf(r.Out, "캐시 경로: %s\n", emptyFallback(result.Dir, "-"))
	_, _ = fmt.Fprintf(r.Out, "파일 수: %d\n", result.Files)
	_, _ = fmt.Fprintf(r.Out, "크기: %s\n", formatBytes(result.Bytes))
}
