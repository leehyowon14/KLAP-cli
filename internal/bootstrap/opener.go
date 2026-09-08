package bootstrap

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func openExternal(target string, kind string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("열 %s이 없습니다", kind)
	}
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", target)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("%s 열기 실패: %w", kind, err)
	}
	return nil
}
