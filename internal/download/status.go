package download

import (
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (*Client) Status(dir string) (app.DownloadStatusResult, error) {
	result := app.DownloadStatusResult{Dir: dir}
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if strings.HasSuffix(entry.Name(), ".part") {
			result.PartialFiles++
			result.PartialBytes += info.Size()
			return nil
		}
		result.Files++
		result.Bytes += info.Size()
		result.Items = append(result.Items, app.DownloadFile{
			Path:       path,
			Bytes:      info.Size(),
			ModifiedAt: info.ModTime(),
		})
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return app.DownloadStatusResult{}, fmt.Errorf("다운로드 폴더 조회 실패: %w", err)
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		return result.Items[j].ModifiedAt.Before(result.Items[i].ModifiedAt)
	})
	if len(result.Items) > 10 {
		result.Items = result.Items[:10]
	}
	return result, nil
}
