package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"github.com/leehyowon14/KLAP-cli/internal/settings"
	"github.com/leehyowon14/KLAP-cli/internal/transcript"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LectureDownloadOptions struct {
	User       UserOption
	Dir        string
	OnProgress func(LectureDownloadProgress)
}

type LectureDownloadAllOptions struct {
	User         UserOption
	CourseFilter string
	Dir          string
	OnProgress   func(LectureDownloadProgress)
	Concurrency  int
	LectureIDs   []string
}

type LectureTranscriptOptions struct {
	Locale     string
	OnProgress func(LectureTranscriptProgress)
}

type LectureDownloadResult struct {
	Path     string
	Bytes    int64
	Lecture  LectureRow
	MediaURL string
}

type DownloadStatusResult struct {
	Dir          string
	Files        int
	Bytes        int64
	PartialFiles int
	PartialBytes int64
	Items        []DownloadFile
}

type DownloadFile struct {
	Path       string
	Bytes      int64
	ModifiedAt time.Time
}

type LectureDownloadItem struct {
	Path    string
	Bytes   int64
	Lecture LectureRow
	Skipped bool
	Err     error
}

type LectureDownloadAllResult struct {
	Items []LectureDownloadItem
}

type LectureTranscriptItem struct {
	Lecture    LectureRow
	InputPath  string
	OutputPath string
	Text       string
	Err        error
}

type LectureTranscriptResult struct {
	Items []LectureTranscriptItem
}

type LectureDownloadProgress struct {
	Lecture      LectureRow
	Path         string
	Stage        LectureTransferStage
	CurrentIndex int
	TotalItems   int
	Bytes        int64
	TotalBytes   int64
	Skipped      bool
	Err          error
}

type LectureTranscriptProgress struct {
	Lecture    LectureRow
	InputPath  string
	OutputPath string
	Stage      LectureTransferStage
	Progress   float64
	Err        error
}

func (s *Service) DownloadStatus(dir string) (DownloadStatusResult, error) {
	dir, err := s.effectiveDownloadDir(dir)
	if err != nil {
		return DownloadStatusResult{}, err
	}
	result := DownloadStatusResult{Dir: dir}
	err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
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
		result.Items = append(result.Items, DownloadFile{
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
		return DownloadStatusResult{}, fmt.Errorf("다운로드 폴더 조회 실패: %w", err)
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		return result.Items[j].ModifiedAt.Before(result.Items[i].ModifiedAt)
	})
	if len(result.Items) > 10 {
		result.Items = result.Items[:10]
	}
	return result, nil
}

func (s *Service) DownloadLecture(ctx context.Context, id string, opts LectureDownloadOptions) (LectureDownloadResult, error) {
	defer s.startCaffeinate(ctx)()
	currentSettings, err := s.loadSettings()
	if err != nil {
		return LectureDownloadResult{}, err
	}

	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return LectureDownloadResult{}, err
	}
	resource, err := s.resolveLectureResource(ctx, studentID, id)
	if err != nil {
		return LectureDownloadResult{}, err
	}
	lectures, err := executeSessionRequest(ctx, s, studentID, &resource.Client, func(client *klas.Client) ([]klas.Lecture, error) {
		return client.Lectures(ctx, resource.Term.Value, resource.Course)
	})
	if err != nil {
		return LectureDownloadResult{}, err
	}

	var matched *klas.Lecture
	for index := range lectures {
		if strings.TrimSpace(lectures[index].ContentID) == resource.Key {
			matched = &lectures[index]
			break
		}
	}
	if matched == nil {
		return LectureDownloadResult{}, fmt.Errorf("강의를 찾을 수 없습니다: %s", id)
	}
	row := LectureRow{
		ID:         resource.ID,
		LegacyID:   resource.LegacyID,
		TermValue:  resource.Term.Value,
		CourseName: resource.Course.Name,
		Lecture:    lectureModel(*matched),
	}
	weekOrder := lectureWeekOrder(lectureModels(lectures), lectureModel(*matched))
	emitLectureDownloadProgress(opts.OnProgress, LectureDownloadProgress{
		Lecture:      row,
		Stage:        LectureStageResolve,
		CurrentIndex: 1,
		TotalItems:   1,
	})

	mediaURL, err := s.media.ResolveLectureMediaURL(ctx, resource.Key)
	if err != nil {
		return LectureDownloadResult{}, err
	}

	dir := strings.TrimSpace(opts.Dir)
	dir, err = s.effectiveDownloadDir(dir)
	if err != nil {
		return LectureDownloadResult{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return LectureDownloadResult{}, fmt.Errorf("다운로드 폴더 생성 실패: %w", err)
	}

	path := s.downloader.VideoPath(dir, resource.Course.Name, lectureFileName(lectureModel(*matched)), mediaURL, weekOrder)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return LectureDownloadResult{}, fmt.Errorf("다운로드 폴더 생성 실패: %w", err)
	}
	emitLectureDownloadProgress(opts.OnProgress, LectureDownloadProgress{
		Lecture:      row,
		Path:         path,
		Stage:        LectureStageDownload,
		CurrentIndex: 1,
		TotalItems:   1,
	})
	bytesWritten, err := s.downloader.DownloadFile(ctx, mediaURL, path, currentSettings.Download.KeepPartial, func(bytesWritten int64, totalBytes int64) {
		emitLectureDownloadProgress(opts.OnProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         path,
			Stage:        LectureStageDownload,
			CurrentIndex: 1,
			TotalItems:   1,
			Bytes:        bytesWritten,
			TotalBytes:   totalBytes,
		})
	})
	if err != nil {
		emitLectureDownloadProgress(opts.OnProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         path,
			Stage:        LectureStageError,
			CurrentIndex: 1,
			TotalItems:   1,
			Bytes:        bytesWritten,
			Err:          err,
		})
		return LectureDownloadResult{}, err
	}
	emitLectureDownloadProgress(opts.OnProgress, LectureDownloadProgress{
		Lecture:      row,
		Path:         path,
		Stage:        LectureStageDone,
		CurrentIndex: 1,
		TotalItems:   1,
		Bytes:        bytesWritten,
		TotalBytes:   bytesWritten,
	})
	return LectureDownloadResult{
		Path:     path,
		Bytes:    bytesWritten,
		Lecture:  row,
		MediaURL: mediaURL,
	}, nil
}

func (s *Service) DownloadAllLectures(ctx context.Context, opts LectureDownloadAllOptions) (LectureDownloadAllResult, error) {
	defer s.startCaffeinate(ctx)()
	currentSettings, err := s.loadSettings()
	if err != nil {
		return LectureDownloadAllResult{}, err
	}

	if strings.TrimSpace(opts.CourseFilter) == "" && len(opts.LectureIDs) == 0 {
		return LectureDownloadAllResult{}, errors.New("전체 다운로드에는 과목명 또는 과목 목록 번호가 필요합니다")
	}

	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}
	stableTermValue, err := stableLectureTermValue(opts.LectureIDs)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}
	var term Term
	if stableTermValue != "" {
		term, client, err = s.termForSyllabus(ctx, studentID, client, stableTermValue)
	} else {
		term, client, err = s.selectedTerm(ctx, studentID, client)
	}
	if err != nil {
		return LectureDownloadAllResult{}, err
	}

	courses, err := selectedCourses(term, opts.CourseFilter)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}

	dir := strings.TrimSpace(opts.Dir)
	dir, err = s.effectiveDownloadDir(dir)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return LectureDownloadAllResult{}, fmt.Errorf("다운로드 폴더 생성 실패: %w", err)
	}

	concurrency, err := s.effectiveDownloadConcurrency(opts.Concurrency)
	if err != nil {
		return LectureDownloadAllResult{}, err
	}
	selectedIDs := make(map[string]struct{}, len(opts.LectureIDs))
	for _, id := range opts.LectureIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			selectedIDs[id] = struct{}{}
		}
	}

	type downloadTask struct {
		Index     int
		Total     int
		Course    Course
		Row       LectureRow
		WeekOrder int
	}
	tasks := make([]downloadTask, 0)
	for _, selectedCourse := range courses {
		lectures, err := executeSessionRequest(ctx, s, studentID, &client, func(client *klas.Client) ([]klas.Lecture, error) {
			return client.Lectures(ctx, term.Value, selectedCourse.Course)
		})
		if err != nil {
			return LectureDownloadAllResult{}, err
		}

		rows := make([]LectureRow, 0, len(lectures))
		for _, lecture := range lectures {
			row, err := newLectureRow(term.Value, selectedCourse, lectureModel(lecture))
			if err != nil {
				return LectureDownloadAllResult{}, err
			}
			rows = append(rows, row)
		}
		weekOrders := lectureRowWeekOrders(rows)
		for _, row := range rows {
			if len(selectedIDs) > 0 {
				_, stableSelected := selectedIDs[row.ID]
				_, legacySelected := selectedIDs[row.LegacyID]
				if !stableSelected && !legacySelected {
					continue
				}
			}
			tasks = append(tasks, downloadTask{
				Course:    selectedCourse.Course,
				Row:       row,
				WeekOrder: weekOrders[row.ID],
			})
		}
	}
	for index := range tasks {
		tasks[index].Index = index + 1
		tasks[index].Total = len(tasks)
	}
	if len(tasks) == 0 {
		return LectureDownloadAllResult{}, nil
	}
	if concurrency > len(tasks) {
		concurrency = len(tasks)
	}

	type downloadTaskResult struct {
		Index int
		Item  LectureDownloadItem
	}
	jobs := make(chan downloadTask)
	results := make(chan downloadTaskResult, len(tasks))
	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for task := range jobs {
			results <- downloadTaskResult{Index: task.Index - 1, Item: downloadLectureTask(ctx, s.media, s.downloader, dir, currentSettings.Download.KeepPartial, task.Index, task.Total, task.Course, task.Row, task.WeekOrder, opts.OnProgress)}
		}
	}
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}
	go func() {
		defer close(jobs)
		for _, task := range tasks {
			select {
			case <-ctx.Done():
				return
			case jobs <- task:
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	result := LectureDownloadAllResult{Items: make([]LectureDownloadItem, len(tasks))}
	for item := range results {
		result.Items[item.Index] = item.Item
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func downloadLectureTask(ctx context.Context, media MediaResolver, downloader Downloader, dir string, keepPartial bool, index int, total int, course Course, row LectureRow, weekOrder int, onProgress func(LectureDownloadProgress)) LectureDownloadItem {
	item := LectureDownloadItem{Lecture: row}
	if strings.TrimSpace(row.Lecture.ContentID) == "" {
		item.Skipped = true
		item.Err = errors.New("KWCommons 콘텐츠 ID가 없습니다")
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Stage:        LectureStageSkip,
			CurrentIndex: index,
			TotalItems:   total,
			Skipped:      true,
			Err:          item.Err,
		})
		return item
	}

	emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
		Lecture:      row,
		Stage:        LectureStageResolve,
		CurrentIndex: index,
		TotalItems:   total,
	})
	mediaURL, err := media.ResolveLectureMediaURL(ctx, row.Lecture.ContentID)
	if err != nil {
		item.Err = err
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Stage:        LectureStageError,
			CurrentIndex: index,
			TotalItems:   total,
			Err:          err,
		})
		return item
	}

	item.Path = downloader.VideoPath(dir, course.Name, lectureFileName(row.Lecture), mediaURL, weekOrder)
	if err := os.MkdirAll(filepath.Dir(item.Path), 0o755); err != nil {
		item.Err = err
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         item.Path,
			Stage:        LectureStageError,
			CurrentIndex: index,
			TotalItems:   total,
			Err:          err,
		})
		return item
	}
	if _, err := os.Stat(item.Path); err == nil {
		item.Skipped = true
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         item.Path,
			Stage:        LectureStageSkip,
			CurrentIndex: index,
			TotalItems:   total,
			Skipped:      true,
		})
		return item
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		item.Err = err
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         item.Path,
			Stage:        LectureStageError,
			CurrentIndex: index,
			TotalItems:   total,
			Err:          err,
		})
		return item
	}

	emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
		Lecture:      row,
		Path:         item.Path,
		Stage:        LectureStageDownload,
		CurrentIndex: index,
		TotalItems:   total,
	})
	bytesWritten, err := downloader.DownloadFile(ctx, mediaURL, item.Path, keepPartial, func(bytesWritten int64, totalBytes int64) {
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         item.Path,
			Stage:        LectureStageDownload,
			CurrentIndex: index,
			TotalItems:   total,
			Bytes:        bytesWritten,
			TotalBytes:   totalBytes,
		})
	})
	item.Bytes = bytesWritten
	if err != nil {
		item.Err = err
		emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
			Lecture:      row,
			Path:         item.Path,
			Stage:        LectureStageError,
			CurrentIndex: index,
			TotalItems:   total,
			Bytes:        bytesWritten,
			Err:          err,
		})
		return item
	}
	emitLectureDownloadProgress(onProgress, LectureDownloadProgress{
		Lecture:      row,
		Path:         item.Path,
		Stage:        LectureStageDone,
		CurrentIndex: index,
		TotalItems:   total,
		Bytes:        bytesWritten,
		TotalBytes:   bytesWritten,
	})
	return item
}

func (s *Service) TranscribeDownloadedLectures(ctx context.Context, items []LectureDownloadItem, opts LectureTranscriptOptions) LectureTranscriptResult {
	defer s.startCaffeinate(ctx)()

	jobs := make([]transcript.Job, 0, len(items))
	rows := make([]LectureRow, 0, len(items))
	failedItems := make([]LectureTranscriptItem, 0)
	for _, item := range items {
		if !LectureDownloadItemNeedsTranscript(item) {
			continue
		}
		outputPath := transcriptPath(item.Path)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			failedItems = append(failedItems, LectureTranscriptItem{
				Lecture:    item.Lecture,
				InputPath:  item.Path,
				OutputPath: outputPath,
				Err:        err,
			})
			emitLectureTranscriptProgress(opts.OnProgress, LectureTranscriptProgress{
				Lecture:    item.Lecture,
				InputPath:  item.Path,
				OutputPath: outputPath,
				Stage:      LectureStageTranscriptError,
				Err:        err,
			})
			continue
		}
		rows = append(rows, item.Lecture)
		jobs = append(jobs, transcript.Job{
			InputPath:         item.Path,
			OutputPath:        outputPath,
			Locale:            opts.Locale,
			ContextualStrings: lectureTranscriptContext(item.Lecture),
		})
		emitLectureTranscriptProgress(opts.OnProgress, LectureTranscriptProgress{
			Lecture:    item.Lecture,
			InputPath:  item.Path,
			OutputPath: outputPath,
			Stage:      LectureStageTranscribe,
		})
	}
	if len(jobs) == 0 {
		return LectureTranscriptResult{Items: failedItems}
	}

	select {
	case <-ctx.Done():
		result := LectureTranscriptResult{Items: append([]LectureTranscriptItem{}, failedItems...)}
		for index, job := range jobs {
			result.Items = append(result.Items, LectureTranscriptItem{
				Lecture:    rows[index],
				InputPath:  job.InputPath,
				OutputPath: job.OutputPath,
				Err:        ctx.Err(),
			})
		}
		return result
	default:
	}

	response, err := s.transcriber.TranscribeWithProgress(ctx, transcript.Request{Jobs: jobs}, func(progress transcript.Progress) {
		index := transcriptJobIndex(jobs, progress.InputPath, progress.OutputPath)
		row := LectureRow{}
		if index >= 0 && index < len(rows) {
			row = rows[index]
		}
		emitLectureTranscriptProgress(opts.OnProgress, LectureTranscriptProgress{
			Lecture:    row,
			InputPath:  progress.InputPath,
			OutputPath: progress.OutputPath,
			Stage:      LectureStageTranscribe,
			Progress:   progress.Progress,
		})
	})
	if err != nil {
		result := LectureTranscriptResult{Items: append([]LectureTranscriptItem{}, failedItems...)}
		for index, job := range jobs {
			item := LectureTranscriptItem{
				Lecture:    rows[index],
				InputPath:  job.InputPath,
				OutputPath: job.OutputPath,
				Err:        err,
			}
			result.Items = append(result.Items, item)
			emitLectureTranscriptProgress(opts.OnProgress, LectureTranscriptProgress{
				Lecture:    item.Lecture,
				InputPath:  item.InputPath,
				OutputPath: item.OutputPath,
				Stage:      LectureStageTranscriptError,
				Err:        err,
			})
		}
		return result
	}

	result := LectureTranscriptResult{Items: append([]LectureTranscriptItem{}, failedItems...)}
	for index, bridgeResult := range response.Results {
		row := LectureRow{}
		if index < len(rows) {
			row = rows[index]
		}
		item := LectureTranscriptItem{
			Lecture:    row,
			InputPath:  bridgeResult.InputPath,
			OutputPath: bridgeResult.OutputPath,
			Text:       bridgeResult.Text,
		}
		stage := LectureStageTranscribed
		if strings.TrimSpace(bridgeResult.Err) != "" {
			item.Err = errors.New(bridgeResult.Err)
			stage = LectureStageTranscriptError
		}
		result.Items = append(result.Items, item)
		emitLectureTranscriptProgress(opts.OnProgress, LectureTranscriptProgress{
			Lecture:    item.Lecture,
			InputPath:  item.InputPath,
			OutputPath: item.OutputPath,
			Stage:      stage,
			Err:        item.Err,
		})
	}
	return result
}

func transcriptJobIndex(jobs []transcript.Job, inputPath string, outputPath string) int {
	for index, job := range jobs {
		if strings.TrimSpace(inputPath) != "" && job.InputPath == inputPath {
			return index
		}
		if strings.TrimSpace(outputPath) != "" && job.OutputPath == outputPath {
			return index
		}
	}
	return -1
}

func transcriptPath(path string) string {
	dir := filepath.Dir(path)
	if filepath.Base(dir) == "video" {
		dir = filepath.Join(filepath.Dir(dir), "transcription")
	}
	ext := filepath.Ext(path)
	if ext == "" {
		return filepath.Join(dir, filepath.Base(path)+".txt")
	}
	return filepath.Join(dir, strings.TrimSuffix(filepath.Base(path), ext)+".txt")
}

func TranscriptPathForDownload(path string) string {
	return transcriptPath(path)
}

func LectureDownloadItemNeedsTranscript(item LectureDownloadItem) bool {
	if item.Err != nil || strings.TrimSpace(item.Path) == "" {
		return false
	}
	if !item.Skipped {
		return true
	}
	_, err := os.Stat(transcriptPath(item.Path))
	return err != nil
}

func lectureTranscriptContext(row LectureRow) []string {
	values := []string{
		"광운대학교",
		"Kwangwoon University",
		"KLAS",
		"이 강의는 한국어와 영어 등 language switching code switching이 포함될 수 있습니다",
		"language switching",
		"code switching",
		row.CourseName,
		row.Lecture.ModuleTitle,
		row.Lecture.Title,
	}
	seen := make(map[string]struct{}, len(values))
	context := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		context = append(context, value)
	}
	return context
}

func emitLectureTranscriptProgress(onProgress func(LectureTranscriptProgress), progress LectureTranscriptProgress) {
	if onProgress != nil {
		onProgress(progress)
	}
}

func defaultLectureDownloadDir() string {
	return settings.DefaultDownloadDir()
}

func (s *Service) effectiveDownloadDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir != "" {
		return dir, nil
	}
	current, err := s.loadSettings()
	if err != nil {
		return "", err
	}
	return current.Download.Dir, nil
}

func (s *Service) effectiveDownloadConcurrency(concurrency int) (int, error) {
	if concurrency > 0 {
		return concurrency, nil
	}
	current, err := s.loadSettings()
	if err != nil {
		return 0, err
	}
	if current.Download.Concurrency <= 0 {
		return settings.DefaultDownloadConcurrency, nil
	}
	return current.Download.Concurrency, nil
}

var lectureWeekPattern = regexp.MustCompile(`([0-9]{1,2})\s*주차`)

func lectureWeekNumber(lecture Lecture) int {
	if week := parsePositiveInt(lecture.WeekNo); week > 0 {
		return week
	}
	match := lectureWeekPattern.FindStringSubmatch(lecture.ModuleTitle)
	if len(match) >= 2 {
		return parsePositiveInt(match[1])
	}
	return 0
}

func lectureWeekOrder(lectures []Lecture, target Lecture) int {
	if order := parsePositiveInt(target.WeeklySeq); order > 0 {
		return order
	}

	week := lectureWeekNumber(target)
	if week <= 0 {
		return 0
	}

	order := 0
	for _, lecture := range lectures {
		if lectureWeekNumber(lecture) != week {
			continue
		}
		order++
		if sameLectureForFilename(lecture, target) {
			return order
		}
	}
	return 0
}

func lectureRowWeekOrders(rows []LectureRow) map[string]int {
	counts := make(map[int]int)
	orders := make(map[string]int, len(rows))
	for _, row := range rows {
		if order := parsePositiveInt(row.Lecture.WeeklySeq); order > 0 {
			orders[row.ID] = order
			week := lectureWeekNumber(row.Lecture)
			if week > 0 && counts[week] < order {
				counts[week] = order
			}
			continue
		}

		week := lectureWeekNumber(row.Lecture)
		if week <= 0 {
			continue
		}
		counts[week]++
		orders[row.ID] = counts[week]
	}
	return orders
}

func sameLectureForFilename(left Lecture, right Lecture) bool {
	if strings.TrimSpace(left.ContentID) != "" && strings.TrimSpace(left.ContentID) == strings.TrimSpace(right.ContentID) {
		return true
	}
	if strings.TrimSpace(left.LearningSeq) != "" && strings.TrimSpace(left.LearningSeq) == strings.TrimSpace(right.LearningSeq) {
		return true
	}
	return strings.TrimSpace(left.ModuleTitle) == strings.TrimSpace(right.ModuleTitle) &&
		strings.TrimSpace(left.Title) == strings.TrimSpace(right.Title)
}

func parsePositiveInt(value string) int {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number <= 0 {
		return 0
	}
	return number
}

func emitLectureDownloadProgress(onProgress func(LectureDownloadProgress), progress LectureDownloadProgress) {
	if onProgress != nil {
		onProgress(progress)
	}
}

type Transcriber interface {
	TranscribeWithProgress(context.Context, transcript.Request, func(transcript.Progress)) (transcript.Response, error)
}
