package app

import (
	"context"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"io"
	"os/exec"
	"runtime"
)

func (s *Service) startCaffeinate(ctx context.Context) func() {
	if runtime.GOOS != "darwin" {
		return func() {}
	}
	current, err := s.loadSettings()
	if err != nil || !settings.DownloadCaffeinateEnabled(current.Download) {
		return func() {}
	}

	cmd := exec.CommandContext(ctx, "caffeinate", "-dims")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return func() {}
	}
	return func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}
}
