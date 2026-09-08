package app

import "context"

// Downloader owns HTTP transfer, partial-file resume and final file promotion.
// Calls are blocking; progress is synchronous and observes completed writes.
type Downloader interface {
	DownloadFile(context.Context, string, string, bool, func(int64, int64)) (int64, error)
}
