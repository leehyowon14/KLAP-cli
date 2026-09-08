package tui

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"sort"
	"strconv"
	"strings"
)

type roomDayOption struct {
	weekday int
	label   string
}

var roomDayOptions = []roomDayOption{
	{weekday: 1, label: "월요일"},
	{weekday: 2, label: "화요일"},
	{weekday: 3, label: "수요일"},
	{weekday: 4, label: "목요일"},
	{weekday: 5, label: "금요일"},
}

type roomAvailableResultsMsg struct {
	results []app.RoomAvailableResult
	err     error
}

func (m model) startRoomFlow() (tea.Model, tea.Cmd) {
	m.active = screenRoomDay
	m.loading = false
	m.err = nil
	m.content = ""
	m.roomDayCursor = 0
	m.roomPeriodCursor = 0
	m.roomDaysSelected = map[int]bool{}
	m.roomPeriodsSelected = map[int]bool{}
	m.roomResults = nil
	m.roomResultPage = 0
	m.roomResultCursor = 0
	return m, nil
}

func (m model) updateRoomDay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ"):
		m.active = screenHome
		m.err = nil
	case key == "up" || keyMatches(key, "k", "ㅏ"):
		if m.roomDayCursor > 0 {
			m.roomDayCursor--
		}
	case key == "down" || keyMatches(key, "j", "ㅓ"):
		if m.roomDayCursor < len(roomDayOptions)-1 {
			m.roomDayCursor++
		}
	case key == " ":
		m.toggleRoomDayCurrent()
	case keyMatches(key, "a", "ㅁ"):
		m.toggleRoomAllDays()
	case key == "enter":
		if len(m.selectedRoomDays()) == 0 {
			m.err = errors.New("요일을 하나 이상 선택하세요")
			return m, nil
		}
		m.err = nil
		m.active = screenRoomPeriod
		m.roomPeriodCursor = 0
		if m.roomPeriodsSelected == nil {
			m.roomPeriodsSelected = map[int]bool{}
		}
	}
	return m, nil
}

func (m model) updateRoomPeriod(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "ctrl+c" || keyMatches(key, "q", "ㅂ"):
		return m, tea.Quit
	case key == "esc" || keyMatches(key, "b", "ㅠ"):
		m.active = screenRoomDay
		m.err = nil
	case key == "up" || keyMatches(key, "k", "ㅏ"):
		if m.roomPeriodCursor > 0 {
			m.roomPeriodCursor--
		}
	case key == "down" || keyMatches(key, "j", "ㅓ"):
		if m.roomPeriodCursor < 7 {
			m.roomPeriodCursor++
		}
	case key == " ":
		m.toggleRoomPeriodCurrent()
	case keyMatches(key, "a", "ㅁ"):
		m.toggleRoomAllPeriods()
	case key == "enter":
		if len(m.selectedRoomPeriods()) == 0 {
			m.err = errors.New("교시를 하나 이상 선택하세요")
			return m, nil
		}
		m.err = nil
		m.active = screenRoomResult
		m.loading = true
		m.content = ""
		m.roomResults = nil
		m.roomResultPage = 0
		m.roomResultCursor = 0
		return m, m.loadRoomAvailableResults(false)
	}
	return m, nil
}

func (m *model) toggleRoomDayCurrent() {
	if m.roomDaysSelected == nil {
		m.roomDaysSelected = map[int]bool{}
	}
	day := roomDayOptions[m.roomDayCursor].weekday
	m.roomDaysSelected[day] = !m.roomDaysSelected[day]
}

func (m *model) toggleRoomAllDays() {
	if m.roomDaysSelected == nil {
		m.roomDaysSelected = map[int]bool{}
	}
	allSelected := true
	for _, day := range roomDayOptions {
		if !m.roomDaysSelected[day.weekday] {
			allSelected = false
			break
		}
	}
	for _, day := range roomDayOptions {
		m.roomDaysSelected[day.weekday] = !allSelected
	}
}

func (m *model) toggleRoomPeriodCurrent() {
	if m.roomPeriodsSelected == nil {
		m.roomPeriodsSelected = map[int]bool{}
	}
	period := m.roomPeriodCursor + 1
	m.roomPeriodsSelected[period] = !m.roomPeriodsSelected[period]
}

func (m *model) toggleRoomAllPeriods() {
	if m.roomPeriodsSelected == nil {
		m.roomPeriodsSelected = map[int]bool{}
	}
	allSelected := true
	for period := 1; period <= 8; period++ {
		if !m.roomPeriodsSelected[period] {
			allSelected = false
			break
		}
	}
	for period := 1; period <= 8; period++ {
		m.roomPeriodsSelected[period] = !allSelected
	}
}

func (m model) selectedRoomDays() []int {
	days := make([]int, 0, len(roomDayOptions))
	for _, day := range roomDayOptions {
		if m.roomDaysSelected[day.weekday] {
			days = append(days, day.weekday)
		}
	}
	return days
}

func (m model) selectedRoomPeriods() []int {
	periods := make([]int, 0, 8)
	for period := 1; period <= 8; period++ {
		if m.roomPeriodsSelected[period] {
			periods = append(periods, period)
		}
	}
	return periods
}

func (m model) loadRoomAvailableResults(refresh bool) tea.Cmd {
	days := m.selectedRoomDays()
	periods := m.selectedRoomPeriods()
	return func() tea.Msg {
		results, err := m.service.RoomAvailabilityForDays(m.ctx, app.RoomAvailabilityForDaysOptions{Days: days, Periods: periods, Refresh: refresh})
		return roomAvailableResultsMsg{results: results, err: err}
	}
}

func (m model) renderRoomDayView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(renderRule(width))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Rooms"))
	b.WriteString("\n")
	b.WriteString("빈 강의실을 조회할 요일을 선택하세요.")
	b.WriteString("\n\n")
	if m.err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(m.err.Error())
		b.WriteString("\n\n")
	}
	for index, option := range roomDayOptions {
		marker := "  "
		if index == m.roomDayCursor {
			marker = "› "
		}
		check := "[ ]"
		if m.roomDaysSelected[option.weekday] {
			check = "[x]"
		}
		line := fmt.Sprintf("%s%s  %s", marker, check, option.label)
		if index == m.roomDayCursor {
			line = menuSelectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderHelpText("↑↓ 이동  |  space 선택  |  a 전체  |  enter 다음  |  b 뒤로  |  q 종료", width))
	return b.String()
}

func (m model) renderRoomPeriodView(width int) string {
	var b strings.Builder
	b.WriteString(m.renderHeader(width))
	b.WriteString("\n")
	b.WriteString(renderRule(width))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Rooms"))
	b.WriteString("\n")
	b.WriteString("빈 강의실을 조회할 교시를 선택하세요.")
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("요일 " + roomSelectedDaysLabel(m.selectedRoomDays())))
	b.WriteString("\n\n")
	if m.err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(m.err.Error())
		b.WriteString("\n\n")
	}
	for period := 1; period <= 8; period++ {
		index := period - 1
		marker := "  "
		if index == m.roomPeriodCursor {
			marker = "› "
		}
		check := "[ ]"
		if m.roomPeriodsSelected[period] {
			check = "[x]"
		}
		line := fmt.Sprintf("%s%s  %d교시", marker, check, period)
		if index == m.roomPeriodCursor {
			line = menuSelectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderHelpText("↑↓ 이동  |  space 선택  |  a 전체  |  enter 조회  |  b 뒤로  |  q 종료", width))
	return b.String()
}

type roomAvailableDisplayRow struct {
	Room   string
	Status string
}

type roomAvailableBuildingGroup struct {
	Building string
	Rows     []roomAvailableDisplayRow
}

func (m model) renderRoomResultPanel(width int) string {
	groups := roomAvailableBuildingGroups(m.roomResults)
	if len(groups) == 0 {
		return emptyStyle.Render("조건에 맞는 빈 강의실이 없습니다") + "\n"
	}
	page := clampInt(m.roomResultPage, 0, len(groups)-1)
	group := groups[page]
	warnings := roomAvailableWarnings(m.roomResults)
	visibleRows := m.visibleRoomResultRows()
	if len(warnings) > 0 {
		visibleRows = maxInt(3, visibleRows-3)
	}
	cursor := clampInt(m.roomResultCursor, 0, maxInt(0, len(group.Rows)-1))
	start := cursor - visibleRows/2
	if start < 0 {
		start = 0
	}
	if start+visibleRows > len(group.Rows) {
		start = maxInt(0, len(group.Rows)-visibleRows)
	}
	end := minInt(len(group.Rows), start+visibleRows)

	var b strings.Builder
	b.WriteString(mutedStyle.Render(fmt.Sprintf("←/→ 건물 이동  %d/%d  %s", page+1, len(groups), group.Building)))
	b.WriteString("\n\n")
	for index := start; index < end; index++ {
		row := group.Rows[index]
		marker := "  "
		if index == cursor {
			marker = "› "
		}
		status := row.Status
		roomWidth := maxInt(8, minInt(28, width-lipgloss.Width(marker)-lipgloss.Width(status)-4))
		room := padRight(truncateText(row.Room, roomWidth), roomWidth)
		line := fmt.Sprintf("%s%s  %s", marker, room, mutedStyle.Render(status))
		if index == cursor {
			line = menuSelectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if start > 0 || end < len(group.Rows) {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  %d-%d / %d", start+1, end, len(group.Rows))))
		b.WriteString("\n")
	}
	if len(warnings) > 0 {
		b.WriteString("\n")
		b.WriteString(warnBadgeStyle.Render("WARN"))
		b.WriteString(fmt.Sprintf(" %d개 과목의 강의시간 조회 실패", len(warnings)))
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) visibleRoomResultRows() int {
	if m.height <= 0 {
		return 14
	}
	return maxInt(5, m.height-11)
}

func formatRoomAvailableResults(results []app.RoomAvailableResult) string {
	if len(results) == 0 {
		return emptyStyle.Render("조건에 맞는 빈 강의실이 없습니다")
	}
	var b strings.Builder
	warnings := make(map[string]struct{})
	for resultIndex, result := range results {
		if resultIndex > 0 {
			b.WriteString("\n")
		}
		status := roomAvailableStatus(result.Weekday, result.Periods)
		b.WriteString(sectionStyle.Render(status))
		if result.Cached {
			b.WriteString(" ")
			b.WriteString(mutedStyle.Render("cache"))
		}
		b.WriteString("\n")
		if len(result.Rooms) == 0 {
			b.WriteString(emptyStyle.Render("조건에 맞는 빈 강의실이 없습니다"))
			b.WriteString("\n")
		} else {
			for index, room := range result.Rooms {
				b.WriteString(fmt.Sprintf("%d. %s  %s\n", index+1, room.Room, mutedStyle.Render(status)))
			}
		}
		for _, warning := range result.Warnings {
			warnings[warning] = struct{}{}
		}
	}
	if len(warnings) > 0 {
		keys := make([]string, 0, len(warnings))
		for warning := range warnings {
			keys = append(keys, warning)
		}
		sort.Strings(keys)
		b.WriteString("\n")
		b.WriteString(warnBadgeStyle.Render("WARN"))
		b.WriteString(fmt.Sprintf(" %d개 과목의 강의시간 조회 실패\n", len(keys)))
		limit := len(keys)
		if limit > 10 {
			limit = 10
		}
		for _, warning := range keys[:limit] {
			b.WriteString("- ")
			b.WriteString(warning)
			b.WriteString("\n")
		}
		if len(keys) > limit {
			b.WriteString(fmt.Sprintf("- ... %d개 생략\n", len(keys)-limit))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func roomAvailableBuildingGroups(results []app.RoomAvailableResult) []roomAvailableBuildingGroup {
	groupRows := make(map[string][]roomAvailableDisplayRow)
	for _, result := range results {
		status := roomAvailableStatus(result.Weekday, result.Periods)
		for _, room := range result.Rooms {
			roomName := strings.TrimSpace(room.Room)
			if roomName == "" {
				continue
			}
			building := roomBuildingName(roomName)
			groupRows[building] = append(groupRows[building], roomAvailableDisplayRow{
				Room:   roomName,
				Status: status,
			})
		}
	}
	buildings := make([]string, 0, len(groupRows))
	for building := range groupRows {
		buildings = append(buildings, building)
	}
	sort.Strings(buildings)
	groups := make([]roomAvailableBuildingGroup, 0, len(buildings))
	for _, building := range buildings {
		rows := groupRows[building]
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].Room == rows[j].Room {
				return rows[i].Status < rows[j].Status
			}
			return rows[i].Room < rows[j].Room
		})
		groups = append(groups, roomAvailableBuildingGroup{Building: building, Rows: rows})
	}
	return groups
}

func roomAvailableWarnings(results []app.RoomAvailableResult) []string {
	seen := make(map[string]struct{})
	for _, result := range results {
		for _, warning := range result.Warnings {
			warning = strings.TrimSpace(warning)
			if warning != "" {
				seen[warning] = struct{}{}
			}
		}
	}
	warnings := make([]string, 0, len(seen))
	for warning := range seen {
		warnings = append(warnings, warning)
	}
	sort.Strings(warnings)
	return warnings
}

func roomBuildingName(room string) string {
	room = strings.TrimSpace(room)
	for _, building := range []string{"화도관", "한천재", "한울관", "참빛관", "옥의관", "연구관", "새빛관", "비마관", "누리관", "기념관"} {
		if strings.HasPrefix(room, building) {
			return building
		}
	}
	for index, r := range room {
		if r >= '0' && r <= '9' {
			if index == 0 {
				return "기타"
			}
			return strings.TrimSpace(room[:index])
		}
	}
	if room == "" {
		return "기타"
	}
	return room
}

func (m *model) moveRoomResultPage(delta int) {
	groups := roomAvailableBuildingGroups(m.roomResults)
	if len(groups) == 0 {
		m.roomResultPage = 0
		m.roomResultCursor = 0
		return
	}
	m.roomResultPage += delta
	if m.roomResultPage < 0 {
		m.roomResultPage = len(groups) - 1
	}
	if m.roomResultPage >= len(groups) {
		m.roomResultPage = 0
	}
	m.roomResultCursor = 0
}

func (m *model) moveRoomResultCursor(delta int) {
	groups := roomAvailableBuildingGroups(m.roomResults)
	if len(groups) == 0 {
		m.roomResultCursor = 0
		return
	}
	page := clampInt(m.roomResultPage, 0, len(groups)-1)
	rows := groups[page].Rows
	if len(rows) == 0 {
		m.roomResultCursor = 0
		return
	}
	m.roomResultCursor += delta
	if m.roomResultCursor < 0 {
		m.roomResultCursor = len(rows) - 1
	}
	if m.roomResultCursor >= len(rows) {
		m.roomResultCursor = 0
	}
}

func roomAvailableStatus(weekday int, periods []int) string {
	return app.RoomWeekdayLabel(weekday) + " " + roomPeriodsLabel(periods) + " 비어있음"
}

func roomPeriodsLabel(periods []int) string {
	normalized := normalizeRoomPeriodsForView(periods)
	if len(normalized) == 0 {
		return "교시 미지정"
	}
	contiguous := true
	for index := 1; index < len(normalized); index++ {
		if normalized[index] != normalized[index-1]+1 {
			contiguous = false
			break
		}
	}
	if contiguous {
		if normalized[0] == normalized[len(normalized)-1] {
			return fmt.Sprintf("%d교시", normalized[0])
		}
		return fmt.Sprintf("%d-%d교시", normalized[0], normalized[len(normalized)-1])
	}
	labels := make([]string, 0, len(normalized))
	for _, period := range normalized {
		labels = append(labels, strconv.Itoa(period))
	}
	return strings.Join(labels, ", ") + "교시"
}

func normalizeRoomPeriodsForView(periods []int) []int {
	normalized := make([]int, 0, len(periods))
	seen := make(map[int]struct{}, len(periods))
	for _, period := range periods {
		if period <= 0 {
			continue
		}
		if _, ok := seen[period]; ok {
			continue
		}
		seen[period] = struct{}{}
		normalized = append(normalized, period)
	}
	sort.Ints(normalized)
	return normalized
}

func roomSelectedDaysLabel(days []int) string {
	if len(days) == 0 {
		return "선택 없음"
	}
	labels := make([]string, 0, len(days))
	for _, weekday := range days {
		labels = append(labels, app.RoomWeekdayLabel(weekday))
	}
	return strings.Join(labels, ", ")
}
