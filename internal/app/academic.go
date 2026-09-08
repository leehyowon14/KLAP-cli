package app

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const academicScheduleURL = "https://biz.kw.ac.kr/undergraduate/schedule.php"

var (
	htmlCommentPattern = regexp.MustCompile(`(?s)<!--.*?-->`)
	h3Pattern          = regexp.MustCompile(`(?is)<h3[^>]*>\s*([0-9]{4})\s*학사일정\s*</h3>`)
	tbodyPattern       = regexp.MustCompile(`(?is)<tbody[^>]*>(.*?)</tbody>`)
	trPattern          = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	tdPattern          = regexp.MustCompile(`(?is)<td[^>]*>(.*?)</td>`)
	htmlTagPattern     = regexp.MustCompile(`(?is)<[^>]+>`)
	spacePattern       = regexp.MustCompile(`[ \t\r\n]+`)
)

type AcademicListOptions struct {
	Year    string
	Refresh bool
}

type AcademicEvent struct {
	Year  string
	Month string
	Date  string
	Title string
	Note  string
}

type AcademicListResult struct {
	Year      string
	SourceURL string
	Events    []AcademicEvent
}

func (s *Service) AcademicList(ctx context.Context, opts AcademicListOptions) (AcademicListResult, error) {
	year := strings.TrimSpace(opts.Year)
	if err := validateAcademicYear(year); err != nil {
		return AcademicListResult{}, err
	}
	if year == "" {
		year = time.Now().In(time.FixedZone("KST", 9*60*60)).Format("2006")
	}

	cacheKey := listCacheKey("academic", "", year, "")
	if !opts.Refresh {
		var cached AcademicListResult
		if _, ok, cacheErr := s.cacheStore.Get(cacheKey, &cached); cacheErr == nil && ok {
			return cached, nil
		}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, academicScheduleURL, nil)
	if err != nil {
		return AcademicListResult{}, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return AcademicListResult{}, fmt.Errorf("학사일정 조회 실패: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return AcademicListResult{}, fmt.Errorf("학사일정 조회 실패: HTTP %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return AcademicListResult{}, err
	}
	events, err := parseAcademicEvents(body, year)
	if err != nil {
		return AcademicListResult{}, err
	}
	result := AcademicListResult{
		Year:      year,
		SourceURL: academicScheduleURL,
		Events:    events,
	}
	_ = s.cacheStore.Set(cacheKey, listCacheTTL(), result)
	return result, nil
}

func parseAcademicEvents(body []byte, year string) ([]AcademicEvent, error) {
	section, err := academicYearSection(string(body), year)
	if err != nil {
		return nil, err
	}

	events := make([]AcademicEvent, 0)
	currentMonth := ""
	for _, tbodyMatch := range tbodyPattern.FindAllStringSubmatch(section, -1) {
		for _, rowMatch := range trPattern.FindAllStringSubmatch(tbodyMatch[1], -1) {
			cells := academicCells(rowMatch[1])
			if len(cells) == 0 {
				continue
			}

			month := currentMonth
			dateIndex := 0
			if isAcademicMonth(cells[0]) {
				month = normalizeAcademicMonth(cells[0])
				currentMonth = month
				dateIndex = 1
			}
			if month == "" || len(cells) <= dateIndex+1 {
				continue
			}

			event := AcademicEvent{
				Year:  year,
				Month: month,
				Date:  cells[dateIndex],
				Title: cells[dateIndex+1],
			}
			if len(cells) > dateIndex+2 {
				event.Note = cells[dateIndex+2]
			}
			if event.Date == "" || event.Title == "" {
				continue
			}
			events = append(events, event)
		}
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("%s 학사일정을 찾을 수 없습니다", year)
	}
	return events, nil
}

func academicYearSection(markup string, year string) (string, error) {
	markup = htmlCommentPattern.ReplaceAllString(markup, "")
	matches := h3Pattern.FindAllStringSubmatchIndex(markup, -1)
	for index, match := range matches {
		if match[2] < 0 || match[3] < 0 || markup[match[2]:match[3]] != year {
			continue
		}

		start := match[1]
		end := len(markup)
		if index+1 < len(matches) {
			end = matches[index+1][0]
		}
		return markup[start:end], nil
	}
	return "", fmt.Errorf("%s 학사일정 섹션을 찾을 수 없습니다", year)
}

func academicCells(rowMarkup string) []string {
	matches := tdPattern.FindAllStringSubmatch(rowMarkup, -1)
	cells := make([]string, 0, len(matches))
	for _, match := range matches {
		text := cleanHTMLText(match[1])
		if text == "" {
			continue
		}
		cells = append(cells, text)
	}
	return cells
}

func cleanHTMLText(markup string) string {
	text := htmlTagPattern.ReplaceAllString(markup, " ")
	text = html.UnescapeString(text)
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = spacePattern.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func normalizeAcademicMonth(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasSuffix(value, "월") {
		return value
	}
	return value + "월"
}

func isAcademicMonth(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasSuffix(value, "월") {
		return false
	}
	value = strings.TrimSuffix(value, "월")
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func validateAcademicYear(year string) error {
	if year == "" {
		return nil
	}
	if len(year) != 4 {
		return errors.New("연도에는 YYYY 형식의 연도가 필요합니다")
	}
	for _, r := range year {
		if r < '0' || r > '9' {
			return errors.New("연도에는 YYYY 형식의 연도가 필요합니다")
		}
	}
	return nil
}
