package macos

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/reminder"
	"runtime"
)

type ReminderBridge struct{ path string }

func NewReminderBridge(path string) ReminderBridge { return ReminderBridge{path: path} }
func (b ReminderBridge) Sync(ctx context.Context, request reminder.SyncRequest) (reminder.SyncResult, error) {
	if runtime.GOOS != "darwin" {
		return reminder.SyncResult{}, errors.New("assignment remind는 현재 macOS에서만 지원합니다")
	}
	var result reminder.SyncResult
	if err := runJSONBridge(ctx, b.path, "reminder", request, &result); err != nil {
		return reminder.SyncResult{}, err
	}
	return result, nil
}
