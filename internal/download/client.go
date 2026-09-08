package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}
type Client struct{ transport HTTPDoer }

func NewClient(transport HTTPDoer) *Client {
	if transport == nil {
		transport = http.DefaultClient
	}
	return &Client{transport: transport}
}

func (c *Client) DownloadFile(ctx context.Context, sourceURL string, path string, keepPartial bool, onProgress func(bytesWritten int64, totalBytes int64)) (int64, error) {
	if _, err := os.Stat(path); err == nil {
		return 0, fmt.Errorf("이미 파일이 있습니다: %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}

	partPath := path + ".part"
	var offset int64
	if stat, err := os.Stat(partPath); err == nil {
		offset = stat.Size()
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("User-Agent", "KLAP-CLI/0.1")
	if offset > 0 {
		request.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	response, err := c.transport.Do(request)
	if err != nil {
		return 0, fmt.Errorf("동영상 다운로드 요청 실패: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	flag := os.O_CREATE | os.O_WRONLY
	if offset > 0 && response.StatusCode == http.StatusPartialContent {
		flag |= os.O_APPEND
	} else {
		offset = 0
		flag |= os.O_TRUNC
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return 0, fmt.Errorf("동영상 다운로드 HTTP 오류: %s", response.Status)
	}
	totalBytes := response.ContentLength
	if totalBytes > 0 {
		totalBytes += offset
	}
	if onProgress != nil {
		onProgress(offset, totalBytes)
	}

	file, err := os.OpenFile(partPath, flag, 0o644)
	if err != nil {
		return 0, fmt.Errorf("임시 파일 열기 실패: %w", err)
	}
	written, copyErr := copyWithProgress(file, response.Body, offset, totalBytes, onProgress)
	closeErr := file.Close()
	if copyErr != nil {
		cleanupPartialDownload(partPath, keepPartial)
		return offset + written, fmt.Errorf("동영상 다운로드 실패: %w", copyErr)
	}
	if closeErr != nil {
		cleanupPartialDownload(partPath, keepPartial)
		return offset + written, fmt.Errorf("임시 파일 닫기 실패: %w", closeErr)
	}

	if err := os.Rename(partPath, path); err != nil {
		return offset + written, fmt.Errorf("다운로드 파일 저장 실패: %w", err)
	}
	return offset + written, nil
}

func cleanupPartialDownload(partPath string, keepPartial bool) {
	if keepPartial {
		return
	}
	_ = os.Remove(partPath)
}

func copyWithProgress(dst io.Writer, src io.Reader, offset int64, totalBytes int64, onProgress func(bytesWritten int64, totalBytes int64)) (int64, error) {
	buffer := make([]byte, 32*1024)
	var written int64
	for {
		nr, readErr := src.Read(buffer)
		if nr > 0 {
			nw, writeErr := dst.Write(buffer[:nr])
			if nw > 0 {
				written += int64(nw)
				if onProgress != nil {
					onProgress(offset+written, totalBytes)
				}
			}
			if writeErr != nil {
				return written, writeErr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return written, nil
			}
			return written, readErr
		}
	}
}
