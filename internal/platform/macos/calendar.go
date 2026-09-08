package macos

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/calendar"
	"runtime"
)

type CalendarBridge struct{ path string }

func NewCalendarBridge(path string) CalendarBridge { return CalendarBridge{path: path} }
func (b CalendarBridge) Sync(ctx context.Context, request calendar.SyncRequest) (calendar.SyncResult, error) {
	if runtime.GOOS != "darwin" {
		return calendar.SyncResult{}, errors.New("academic calendar sync는 현재 macOS에서만 지원합니다")
	}
	var result calendar.SyncResult
	if err := runJSONBridge(ctx, b.path, "calendar", request, &result); err != nil {
		return calendar.SyncResult{}, err
	}
	return result, nil
}
