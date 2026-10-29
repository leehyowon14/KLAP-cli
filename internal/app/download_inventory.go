package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

type DownloadInventoryItem struct {
	ID, Path string
	Bytes    int64
}

func (s *Service) DownloadInventory(ctx context.Context, opts LectureDownloadAllOptions) ([]DownloadInventoryItem, error) {
	rows, err := s.LectureList(ctx, LectureListOptions{User: opts.User})
	if err != nil {
		return nil, err
	}
	groups := map[string][]LectureRow{}
	for _, row := range rows {
		groups[row.CourseName] = append(groups[row.CourseName], row)
	}
	found := []DownloadInventoryItem{}
	for _, group := range groups {
		orders := lectureRowWeekOrders(group)
		for _, row := range group {
			path := s.downloader.VideoPath(opts.Dir, row.CourseName, lectureFileName(row.Lecture), "", orders[row.ID])
			if existing, size := existingDownloadFile(path); existing != "" {
				found = append(found, DownloadInventoryItem{ID: row.ID, Path: existing, Bytes: size})
			}
		}
	}
	return found, nil
}

func existingDownloadFile(path string) (string, int64) {
	for _, candidate := range []string{path, filepath.Join(filepath.Dir(filepath.Dir(path)), "Video", filepath.Base(path)), filepath.Join(filepath.Dir(filepath.Dir(path)), "video", filepath.Base(path))} {
		files, _ := os.ReadDir(filepath.Dir(candidate))
		for _, file := range files {
			ext := strings.ToLower(filepath.Ext(file.Name()))
			if ext != ".mp4" && ext != ".mov" && ext != ".m4v" && ext != ".webm" && ext != ".mkv" {
				continue
			}
			if strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())) != strings.TrimSuffix(filepath.Base(candidate), filepath.Ext(candidate)) {
				continue
			}
			info, e := file.Info()
			if e == nil && info.Mode().IsRegular() && info.Size() > 0 {
				return filepath.Join(filepath.Dir(candidate), file.Name()), info.Size()
			}
		}
	}
	return "", 0
}
