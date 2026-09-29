package klas

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

type BoardAttachment struct {
	FileSN   string `json:"FileSN"`
	Name     string `json:"Name"`
	Size     int64  `json:"Size"`
	download string
}

func (c *Client) BoardAttachments(ctx context.Context, attachmentID string) ([]BoardAttachment, error) {
	if strings.TrimSpace(attachmentID) == "" {
		return []BoardAttachment{}, nil
	}
	b, e := c.do(ctx, http.MethodPost, "/common/file/UploadFileList.do", map[string]string{"storageId": "CLS_BOARD", "attachId": attachmentID})
	if e != nil {
		return nil, e
	}
	var rows []struct {
		FileSN   flexibleString `json:"fileSn"`
		Name     string         `json:"fileName"`
		Size     int64          `json:"fileSize"`
		Download string         `json:"download"`
	}
	if e = decodeResponseJSON(b, &rows); e != nil {
		return nil, fmt.Errorf("첨부파일 목록 파싱 실패: %w", e)
	}
	result := make([]BoardAttachment, 0, len(rows))
	for _, r := range rows {
		if r.FileSN == "" || r.Name == "" || r.Download == "" {
			return nil, errors.New("첨부파일 정보가 누락되었습니다")
		}
		result = append(result, BoardAttachment{r.FileSN.String(), r.Name, r.Size, r.Download})
	}
	return result, nil
}

// DownloadBoardAttachment accepts only a freshly resolved attachment; session cookies stay in CLI.
func (c *Client) DownloadBoardAttachment(ctx context.Context, file BoardAttachment, directory string) (path string, err error) {
	relative, e := url.Parse(file.download)
	if e != nil {
		return "", e
	}
	endpoint := c.baseURL.ResolveReference(relative)
	if endpoint.Scheme != c.baseURL.Scheme || endpoint.Host != c.baseURL.Host || !strings.HasPrefix(endpoint.Path, "/common/file/DownloadFile/") {
		return "", errors.New("잘못된 첨부파일 주소입니다")
	}
	client := *c.httpClient
	client.Timeout = 3 * time.Minute
	client.CheckRedirect = func(r *http.Request, via []*http.Request) error {
		if len(via) >= 5 || r.URL.Scheme != endpoint.Scheme || r.URL.Host != endpoint.Host {
			return errors.New("허용되지 않는 다운로드 이동입니다")
		}
		return nil
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if e != nil {
		return "", e
	}
	resp, e := client.Do(req)
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return "", ErrSessionExpired
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("첨부파일 다운로드 HTTP %d", resp.StatusCode)
	}
	reader := bufio.NewReader(resp.Body)
	head, _ := reader.Peek(512)
	if looksLikeLoginPageHTML(head) {
		return "", ErrSessionExpired
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if (strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/json")) && !strings.Contains(strings.ToLower(resp.Header.Get("Content-Disposition")), "attachment") {
		return "", errors.New("파일 대신 KLAS 오류 응답을 받았습니다")
	}
	if e = os.MkdirAll(directory, 0700); e != nil {
		return "", e
	}
	folder, e := os.MkdirTemp(directory, "attachment-")
	if e != nil {
		return "", e
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(folder)
		}
	}()
	path = filepath.Join(folder, safeAttachmentName(file.Name))
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return "", e
	}
	n, copyErr := io.Copy(f, reader)
	closeErr := f.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if (resp.ContentLength >= 0 && n != resp.ContentLength) || (file.Size > 0 && n != file.Size) {
		return "", errors.New("첨부파일 다운로드가 중간에 끊겼습니다")
	}
	return path, nil
}
func safeAttachmentName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == ':' {
			return '_'
		}
		return r
	}, name)
	if strings.TrimSpace(name) == "" || name == "." || name == ".." || name == "/" {
		return "첨부파일"
	}
	return name
}
