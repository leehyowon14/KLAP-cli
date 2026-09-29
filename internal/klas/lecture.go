package klas

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/kwcommons"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Lecture struct {
	ViewerSupported  bool
	FirstStartedAt   *time.Time
	FirstCompletedAt *time.Time
	ContentID        string
	PlayURL          string
	LearningSeq      string
	FileID           string
	WeekNo           string
	WeeklySeq        string
	ModuleTitle      string
	Title            string
	Progress         string
	AchievedTime     string
	RequiredTime     string
	StartAt          *time.Time
	EndAt            *time.Time
	Raw              lectureListItem `json:"-"`
}

type LectureProgress struct {
	TotalTime string
	PTime     string
	Progress  float64
	Completed bool
}

type lectureListItem struct {
	FirstEdu     string         `json:"firstEdu"`
	FirstEnd     string         `json:"firstEnd"`
	GroupCode    string         `json:"grcode"`
	SubjectID    string         `json:"subj"`
	Year         string         `json:"year"`
	Hakgi        string         `json:"hakgi"`
	Bunban       string         `json:"bunban"`
	Module       flexibleString `json:"module"`
	Lesson       flexibleString `json:"lesson"`
	OID          string         `json:"oid"`
	PTime        flexibleString `json:"ptime"`
	TotalTime    flexibleString `json:"totalTime"`
	LearnTime    flexibleString `json:"learnTime"`
	AchivTime    flexibleString `json:"achivTime"`
	RcognTime    flexibleString `json:"rcognTime"`
	TotRcognTime flexibleString `json:"totRcognTime"`
	TotAchivTime flexibleString `json:"totAchivTime"`
	WeekNo       flexibleString `json:"weekNo"`
	WeeklySeq    flexibleString `json:"weeklyseq"`
	LearningSeq  flexibleString `json:"lrnSn"`
	FileID       flexibleString `json:"fileId"`
	IsPreview    string         `json:"ispreview"`
	Evaluation   string         `json:"evltnSe"`
	Title        string         `json:"sbjt"`
	ModuleTitle  string         `json:"moduletitle"`
	Progress     flexibleString `json:"prog"`
	StartDate    string         `json:"startDate"`
	EndDate      string         `json:"endDate"`
	StartY       string         `json:"sdateY"`
	StartH       string         `json:"sdateH"`
	StartM       string         `json:"sdateM"`
	EndY         string         `json:"edateY"`
	EndH         string         `json:"edateH"`
	EndM         string         `json:"edateM"`
	Starting     string         `json:"starting"`
	MVPLink      string         `json:"mvpLink"`
}

func (c *Client) Lectures(ctx context.Context, yearHakgi string, course Course) ([]Lecture, error) {
	if err := c.SetCourseContext(ctx, yearHakgi, course); err != nil {
		return nil, err
	}

	body, err := c.do(ctx, http.MethodPost, "/std/lis/evltn/SelectOnlineCntntsStdList.do", map[string]any{})
	if err != nil {
		return nil, err
	}

	var response []lectureListItem
	if err := decodeResponseJSON(body, &response); err != nil {
		return nil, fmt.Errorf("강의 목록 응답 파싱 실패: %w", err)
	}

	lectures := make([]Lecture, 0, len(response))
	for _, item := range response {
		if strings.EqualFold(strings.TrimSpace(item.Evaluation), "proj") {
			continue
		}
		if strings.TrimSpace(item.SubjectID) != "" && strings.TrimSpace(course.Value) != "" && strings.TrimSpace(item.SubjectID) != strings.TrimSpace(course.Value) {
			continue
		}

		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = "제목 없음"
		}
		contentID := kwcommons.ExtractKWCommonsContentID(item.MVPLink, item.Starting)
		requiredTime, achievedTime := lectureListTimes(item)
		_, viewerErr := lectureViewerForm(Lecture{Raw: item})
		lectures = append(lectures, Lecture{
			ViewerSupported:  viewerErr == nil,
			FirstStartedAt:   parseKlasDateTime(item.FirstEdu),
			FirstCompletedAt: parseKlasDateTime(item.FirstEnd),
			ContentID:        contentID,
			PlayURL:          kwcommons.NormalizePlayURL(firstNonEmpty(item.MVPLink, item.Starting), contentID),
			LearningSeq:      item.LearningSeq.String(),
			FileID:           item.FileID.String(),
			WeekNo:           item.WeekNo.String(),
			WeeklySeq:        item.WeeklySeq.String(),
			ModuleTitle:      strings.TrimSpace(item.ModuleTitle),
			Title:            title,
			Progress:         item.Progress.String(),
			AchievedTime:     achievedTime,
			RequiredTime:     requiredTime,
			StartAt:          parseLectureDateTime(item.StartDate, item.StartY, item.StartH, item.StartM),
			EndAt:            parseLectureDateTime(item.EndDate, item.EndY, item.EndH, item.EndM),
			Raw:              item,
		})
	}
	return lectures, nil
}

func (c *Client) LectureKey(ctx context.Context, lecture Lecture) (string, error) {
	form, err := lectureViewerForm(lecture)
	if err != nil {
		return "", err
	}

	body, err := c.doForm(ctx, "/spv/lis/lctre/viewer/LctreCntntsViewSpvPage.do", form, true)
	if err != nil {
		return "", err
	}

	match := lectureKeyPattern.FindSubmatch(body)
	if len(match) < 2 {
		return "", schemaError(errors.New("강의 뷰어 응답에서 lecKey를 찾지 못했습니다"))
	}
	return string(match[1]), nil
}

func (c *Client) CheckLectureView(ctx context.Context, lecture Lecture, lecKey string) error {
	form, err := lectureProgressForm(lecture, lecKey)
	if err != nil {
		return err
	}
	_, err = c.doForm(ctx, "/spv/lis/lctre/viewer/ChkLctreCntntsView.do", form, false)
	return err
}

func (c *Client) UpdateLectureProgress(ctx context.Context, lecture Lecture, lecKey string) (LectureProgress, error) {
	form, err := lectureProgressForm(lecture, lecKey)
	if err != nil {
		return LectureProgress{}, err
	}

	body, err := c.doForm(ctx, "/spv/lis/lctre/viewer/UpdateProgress.do", form, false)
	if err != nil {
		return LectureProgress{}, err
	}
	return parseLectureProgress(body)
}

func (c *Client) SaveLectureLearningStatus(ctx context.Context, lecture Lecture, lrnStatus string) (LectureProgress, error) {
	payload, err := lectureLearningStatusPayload(lecture, lrnStatus)
	if err != nil {
		return LectureProgress{}, err
	}

	body, err := c.do(ctx, http.MethodPost, "/std/lis/evltn/SaveLrnStatus.do", payload)
	if err != nil {
		return LectureProgress{}, err
	}
	if strings.Trim(strings.TrimSpace(string(body)), `"`) != "Y" {
		return LectureProgress{}, &Error{Kind: ErrorRemoteBusiness, Err: errors.New("학습활동 수강 상태 저장에 실패했습니다")}
	}

	requiredTime := strings.TrimSpace(lecture.RequiredTime)
	return LectureProgress{
		TotalTime: firstNonEmpty(requiredTime, strings.TrimSpace(lecture.Progress)),
		PTime:     requiredTime,
		Progress:  100,
		Completed: true,
	}, nil
}

func lectureViewerForm(lecture Lecture) (url.Values, error) {
	item := lecture.Raw
	required := map[string]string{
		"grcode":    item.GroupCode,
		"subj":      item.SubjectID,
		"year":      item.Year,
		"hakgi":     item.Hakgi,
		"bunban":    item.Bunban,
		"module":    item.Module.String(),
		"oid":       item.OID,
		"ptime":     item.PTime.String(),
		"weeklyseq": item.WeekNo.String(),
		"lesson":    item.Lesson.String(),
	}
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("강의 수강 필드가 없습니다: %s", key)
		}
	}

	values := url.Values{}
	values.Set("grcode", item.GroupCode)
	values.Set("subj", item.SubjectID)
	values.Set("year", item.Year)
	values.Set("hakgi", item.Hakgi)
	values.Set("bunban", item.Bunban)
	values.Set("module", item.Module.String())
	values.Set("oid", item.OID)
	values.Set("ptime", item.PTime.String())
	values.Set("weeklyseq", item.WeekNo.String())
	values.Set("weeklysubseq", item.WeeklySeq.String())
	values.Set("totalTime", item.TotalTime.String())
	values.Set("prog", item.Progress.String())
	values.Set("lesson", item.Lesson.String())
	values.Set("profYN", "Y")
	values.Set("previewYN", firstNonEmpty(item.IsPreview, "N"))
	values.Set("late", "N")
	return values, nil
}

func lectureProgressForm(lecture Lecture, lecKey string) (url.Values, error) {
	lecKey = strings.TrimSpace(lecKey)
	if lecKey == "" {
		return nil, errors.New("lecKey가 없습니다")
	}

	viewerForm, err := lectureViewerForm(lecture)
	if err != nil {
		return nil, err
	}
	values := url.Values{}
	for _, key := range []string{"grcode", "subj", "year", "hakgi", "bunban", "module", "oid", "ptime", "weeklyseq", "weeklysubseq", "lesson"} {
		values.Set(key, viewerForm.Get(key))
	}
	values.Set("lecKey", lecKey)
	return values, nil
}

func lectureLearningStatusPayload(lecture Lecture, lrnStatus string) (map[string]any, error) {
	item := lecture.Raw
	required := map[string]string{
		"grcode": item.GroupCode,
		"subj":   item.SubjectID,
		"year":   item.Year,
		"hakgi":  item.Hakgi,
		"bunban": item.Bunban,
		"lrnSn":  item.LearningSeq.String(),
	}
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("학습활동 수강 필드가 없습니다: %s", key)
		}
	}
	if strings.TrimSpace(lrnStatus) == "" {
		return nil, errors.New("학습활동 수강 상태가 없습니다")
	}

	return map[string]any{
		"grcode":    item.GroupCode,
		"subj":      item.SubjectID,
		"year":      item.Year,
		"hakgi":     item.Hakgi,
		"bunban":    item.Bunban,
		"lrnSn":     item.LearningSeq.String(),
		"lrnStatus": lrnStatus,
	}, nil
}

func parseLectureProgress(body []byte) (LectureProgress, error) {
	var root map[string]any
	if err := decodeResponseJSON(body, &root); err != nil {
		return LectureProgress{}, fmt.Errorf("수강 진도 응답 파싱 실패: %w", err)
	}

	source := root
	if nested, ok := root["data"].(map[string]any); ok {
		source = nested
	}
	progressText := rowString(source, "prog")
	if progressText == "" {
		return LectureProgress{}, schemaError(errors.New("수강 진도 응답에 prog가 없습니다"))
	}
	progress, err := strconv.ParseFloat(progressText, 64)
	if err != nil {
		return LectureProgress{}, schemaError(fmt.Errorf("수강 진도 prog 파싱 실패: %w", err))
	}
	return LectureProgress{
		TotalTime: rowString(source, "totalTime"),
		PTime:     rowString(source, "ptime"),
		Progress:  progress,
		Completed: progress >= 100,
	}, nil
}

func parseLectureDateTime(dateTime string, compactDate string, hour string, minute string) *time.Time {
	if parsed := parseKoreanDateTime(dateTime); parsed != nil {
		return parsed
	}

	compactDate = strings.TrimSpace(compactDate)
	if compactDate == "" {
		return nil
	}
	hour = firstNonEmpty(hour, "00")
	minute = firstNonEmpty(minute, "00")
	value := compactDate + hour + minute

	location, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		location = time.FixedZone("KST", 9*60*60)
	}

	parsed, err := time.ParseInLocation("200601021504", value, location)
	if err != nil {
		return nil
	}
	return &parsed
}

// List response totalTime can be content duration even when prog and achivTime
// are zero. It is not the same field semantics as UpdateProgress.totalTime.
func lectureListTimes(item lectureListItem) (required, achieved string) {
	return firstNonEmpty(item.RcognTime.String(), item.TotRcognTime.String(), item.PTime.String()),
		firstNonEmpty(item.AchivTime.String(), item.LearnTime.String(), item.TotAchivTime.String())
}
