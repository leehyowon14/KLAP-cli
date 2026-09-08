package tui

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (m *academicScreenModel) moveAcademicMonth(delta int) {
	m.academicMonth += delta
	if m.academicMonth < 1 {
		m.academicMonth = 12
	}
	if m.academicMonth > 12 {
		m.academicMonth = 1
	}
	m.academicCursor = 0
}

func (m *academicScreenModel) moveAcademicCursor(delta int) {
	events := academicMonthEvents(m.academicResult, m.academicMonth)
	if len(events) == 0 {
		m.academicCursor = 0
		return
	}
	m.academicCursor += delta
	if m.academicCursor < 0 {
		m.academicCursor = 0
	}
	if m.academicCursor >= len(events) {
		m.academicCursor = len(events) - 1
	}
}

func (m academicScreenModel) View(width, height int, status string, now time.Time) string {
	month := m.academicMonth
	if month < 1 || month > 12 {
		month = defaultAcademicMonth(m.academicResult.Events, now)
	}
	events := academicMonthEvents(m.academicResult, month)
	selected := app.AcademicEvent{}
	if len(events) > 0 {
		if m.academicCursor < 0 {
			m.academicCursor = 0
		}
		if m.academicCursor >= len(events) {
			m.academicCursor = len(events) - 1
		}
		selected = events[m.academicCursor]
	}
	var b strings.Builder
	b.WriteString(mutedStyle.Render(fmt.Sprintf("%s년 %d월", m.academicResult.Year, month)))
	b.WriteString("\n\n")
	b.WriteString(renderAcademicMonthCalendar(m.academicResult, month, width, selected))
	b.WriteString("\n")
	b.WriteString(renderAcademicEventList(events, m.academicCursor, width, visibleBodyRows(height, 10)))
	if status != "" {
		b.WriteString("\n")
		b.WriteString(footerStyle.Render(status))
		b.WriteString("\n")
	}
	return b.String()
}

func renderAcademicMonthCalendar(result app.AcademicListResult, month int, width int, selected app.AcademicEvent) string {
	year, err := strconv.Atoi(strings.TrimSpace(result.Year))
	if err != nil || month < 1 || month > 12 {
		return emptyStyle.Render("학사일정 연도를 확인할 수 없습니다") + "\n"
	}
	eventsByDay := make(map[int][]app.AcademicEvent)
	for _, event := range result.Events {
		startAt, endAt, ok := app.AcademicEventRange(event)
		if !ok {
			continue
		}
		for dayAt := startAt; dayAt.Before(endAt); dayAt = dayAt.AddDate(0, 0, 1) {
			if dayAt.Year() != year || int(dayAt.Month()) != month {
				continue
			}
			eventsByDay[dayAt.Day()] = append(eventsByDay[dayAt.Day()], event)
		}
	}
	selectedDays := selectedAcademicDays(selected, year, month)

	var b strings.Builder
	b.WriteString("월  화  수  목  금  토  일\n")
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	offset := int(first.Weekday()+6) % 7
	days := daysInMonth(year, month)
	for index := 0; index < offset; index++ {
		b.WriteString("    ")
	}
	for day := 1; day <= days; day++ {
		label := fmt.Sprintf("%2d", day)
		if selectedDays[day] {
			label = selectedDayStyle.Render(label)
		} else if len(eventsByDay[day]) > 0 {
			label = warnTextStyle.Render(label)
		}
		b.WriteString(label)
		if (offset+day)%7 == 0 {
			b.WriteString("\n")
		} else {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n")
	return b.String()
}

func academicMonthEvents(result app.AcademicListResult, month int) []app.AcademicEvent {
	year, err := strconv.Atoi(strings.TrimSpace(result.Year))
	if err != nil || month < 1 || month > 12 {
		return nil
	}
	events := make([]app.AcademicEvent, 0)
	for _, event := range result.Events {
		startAt, endAt, ok := app.AcademicEventRange(event)
		if !ok {
			continue
		}
		monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		monthEnd := monthStart.AddDate(0, 1, 0)
		if endAt.After(monthStart) && startAt.Before(monthEnd) {
			events = append(events, event)
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		left, _, leftOK := app.AcademicEventRange(events[i])
		right, _, rightOK := app.AcademicEventRange(events[j])
		if !leftOK || !rightOK {
			return events[i].Title < events[j].Title
		}
		if left.Equal(right) {
			return events[i].Title < events[j].Title
		}
		return left.Before(right)
	})
	return events
}

func renderAcademicEventList(events []app.AcademicEvent, cursor int, width int, visibleRows int) string {
	if len(events) == 0 {
		return emptyStyle.Render("이 달의 학사일정이 없습니다") + "\n"
	}
	lines := make([]string, 0, len(events))
	for index, event := range events {
		marker := "  "
		if index == cursor {
			marker = "› "
		}
		line := fmt.Sprintf("%s%s  %s", marker, academicEventRangeLabel(event), truncateText(event.Title, maxInt(16, minInt(72, width-18))))
		if index == cursor {
			line = menuSelectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return renderWindowedLines(lines, cursor, visibleRows)
}

func academicEventRangeLabel(event app.AcademicEvent) string {
	startAt, endAt, ok := app.AcademicEventRange(event)
	if !ok {
		return strings.TrimSpace(event.Date)
	}
	endInclusive := endAt.AddDate(0, 0, -1)
	if startAt.Equal(endInclusive) {
		return fmt.Sprintf("%d일", startAt.Day())
	}
	if startAt.Month() == endInclusive.Month() {
		return fmt.Sprintf("%d일-%d일", startAt.Day(), endInclusive.Day())
	}
	return fmt.Sprintf("%d/%d-%d/%d", int(startAt.Month()), startAt.Day(), int(endInclusive.Month()), endInclusive.Day())
}

func selectedAcademicDays(event app.AcademicEvent, year int, month int) map[int]bool {
	days := make(map[int]bool)
	startAt, endAt, ok := app.AcademicEventRange(event)
	if !ok {
		return days
	}
	for dayAt := startAt; dayAt.Before(endAt); dayAt = dayAt.AddDate(0, 0, 1) {
		if dayAt.Year() == year && int(dayAt.Month()) == month {
			days[dayAt.Day()] = true
		}
	}
	return days
}

func defaultAcademicMonth(events []app.AcademicEvent, now time.Time) int {
	if len(events) == 0 {
		return int(now.Month())
	}
	for _, event := range events {
		dueAt, ok := academicEventDate(event)
		if ok && !dueAt.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)) {
			return int(dueAt.Month())
		}
	}
	if dueAt, ok := academicEventDate(events[0]); ok {
		return int(dueAt.Month())
	}
	return int(now.Month())
}

var tuiAcademicDatePattern = regexp.MustCompile(`([0-9]{1,2})\s*[./]\s*([0-9]{1,2})|([0-9]{1,2})\s*일|([0-9]{1,2})\s*\(`)

func academicEventDate(event app.AcademicEvent) (time.Time, bool) {
	year, err := strconv.Atoi(strings.TrimSpace(event.Year))
	if err != nil {
		return time.Time{}, false
	}
	month, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(event.Month), "월"))
	if err != nil {
		return time.Time{}, false
	}
	match := tuiAcademicDatePattern.FindStringSubmatch(strings.TrimSpace(event.Date))
	if len(match) == 0 {
		return time.Time{}, false
	}
	dayText := firstNonEmptyString(match[2], match[3], match[4])
	if match[1] != "" && match[2] != "" {
		if parsedMonth, monthErr := strconv.Atoi(match[1]); monthErr == nil {
			month = parsedMonth
		}
	}
	day, err := strconv.Atoi(dayText)
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), true
}

func daysInMonth(year int, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local).Day()
}

type academicScreenModel struct {
	academicResult app.AcademicListResult
	academicMonth  int
	academicCursor int
}

func (m *academicScreenModel) Loaded(result app.AcademicListResult, active bool, now time.Time) {
	m.academicResult = result
	if active {
		m.academicMonth = defaultAcademicMonth(result.Events, now)
		m.academicCursor = 0
	} else if m.academicMonth == 0 {
		m.academicMonth = defaultAcademicMonth(result.Events, now)
	}
}
func (m *academicScreenModel) Reset() { *m = academicScreenModel{} }
func (m *academicScreenModel) Update(msg tea.Msg) (childAction, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return childAction{}, false
	}
	switch {
	case key.String() == "up" || keyMatches(key.String(), "k", "ㅏ"):
		m.moveAcademicCursor(-1)
	case key.String() == "down" || keyMatches(key.String(), "j", "ㅓ"):
		m.moveAcademicCursor(1)
	case key.String() == "left":
		m.moveAcademicMonth(-1)
	case key.String() == "right":
		m.moveAcademicMonth(1)
	default:
		return childAction{}, false
	}
	return childAction{}, true
}

type academicScreenService interface {
	AcademicList(context.Context, app.AcademicListOptions) (app.AcademicListResult, error)
}

func loadAcademic(ctx context.Context, service academicScreenService, refresh, prefetch bool) tea.Cmd {
	return func() tea.Msg {
		result, err := service.AcademicList(ctx, app.AcademicListOptions{Refresh: refresh})
		return loadMsg{screen: screenAcademic, prefetch: prefetch, academic: result, err: err}
	}
}
func (m model) renderAcademicCalendarPanel(width int) string {
	return m.academic.View(width, m.height, m.syncStatus, time.Now())
}
