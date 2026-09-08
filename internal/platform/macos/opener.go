package macos

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

// OpenExternal preserves the desktop launch contract: return once the helper
// starts, not when the browser quits. ProcessRunner still reaps the helper and
// honours caller cancellation; launch failures retain their typed cause.
func OpenExternal(ctx context.Context, target, kind string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("열 %s이 없습니다", kind)
	}
	_, err := (ProcessRunner{}).Start(ctx, externalSpec(runtime.GOOS, target))
	if err != nil {
		return fmt.Errorf("%s 열기 실패: %w", kind, err)
	}
	return nil
}

func externalSpec(platform, target string) ProcessSpec {
	switch platform {
	case "darwin":
		return ProcessSpec{Name: "open", Args: []string{target}}
	case "windows":
		return ProcessSpec{Name: "rundll32", Args: []string{"url.dll,FileProtocolHandler", target}}
	default:
		return ProcessSpec{Name: "xdg-open", Args: []string{target}}
	}
}
