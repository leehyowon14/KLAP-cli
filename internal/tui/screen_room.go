package tui

import (
	"context"
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

func (m *roomScreenModel) toggleRoomDayCurrent() {
	if m.roomDaysSelected == nil {
		m.roomDaysSelected = map[int]bool{}
	}
	day := roomDayOptions[m.roomDayCursor].weekday
	m.roomDaysSelected[day] = !m.roomDaysSelected[day]
}

func (m *roomScreenModel) toggleRoomAllDays() {
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

func (m *roomScreenModel) toggleRoomPeriodCurrent() {
	if m.roomPeriodsSelected == nil {
		m.roomPeriodsSelected = map[int]bool{}
	}
	period := m.roomPeriodCursor + 1
	m.roomPeriodsSelected[period] = !m.roomPeriodsSelected[period]
}

func (m *roomScreenModel) toggleRoomAllPeriods() {
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

func (m roomScreenModel) selectedRoomDays() []int {
	days := make([]int, 0, len(roomDayOptions))
	for _, day := range roomDayOptions {
		if m.roomDaysSelected[day.weekday] {
			days = append(days, day.weekday)
		}
	}
	return days
}

func (m roomScreenModel) selectedRoomPeriods() []int {
	periods := make([]int, 0, 8)
	for period := 1; period <= 8; period++ {
		if m.roomPeriodsSelected[period] {
			periods = append(periods, period)
		}
	}
	return periods
}

func (m roomScreenModel) Load(ctx context.Context, service roomScreenService, refresh bool) tea.Cmd {
	days := m.selectedRoomDays()
	periods := m.selectedRoomPeriods()
	return func() tea.Msg {
		results, err := service.RoomAvailabilityForDays(ctx, app.RoomAvailabilityForDaysOptions{Days: days, Periods: periods, Refresh: refresh})
		return roomAvailableResultsMsg{results: results, err: err}
	}
}

func (m roomScreenModel) renderRoomDayView(width int, err error) string {
	var b strings.Builder
	b.WriteString(renderHeaderTitle(width, "Rooms"))
	b.WriteString("\n")
	b.WriteString(renderRule(width))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Rooms"))
	b.WriteString("\n")
	b.WriteString("빈 강의실을 조회할 요일을 선택하세요.")
	b.WriteString("\n\n")
	if err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(err.Error())
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

func (m roomScreenModel) renderRoomPeriodView(width int, err error) string {
	var b strings.Builder
	b.WriteString(renderHeaderTitle(width, "Rooms"))
	b.WriteString("\n")
	b.WriteString(renderRule(width))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Rooms"))
	b.WriteString("\n")
	b.WriteString("빈 강의실을 조회할 교시를 선택하세요.")
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("요일 " + roomSelectedDaysLabel(m.selectedRoomDays())))
	b.WriteString("\n\n")
	if err != nil {
		b.WriteString(errorStyle.Render("ERROR"))
		b.WriteString(" ")
		b.WriteString(err.Error())
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

func (m roomScreenModel) renderRoomResultPanel(width, height int) string {
	groups := roomAvailableBuildingGroups(m.roomResults)
	if len(groups) == 0 {
		return emptyStyle.Render("조건에 맞는 빈 강의실이 없습니다") + "\n"
	}
	page := clampInt(m.roomResultPage, 0, len(groups)-1)
	group := groups[page]
	warnings := roomAvailableWarnings(m.roomResults)
	visibleRows := visibleRoomResultRows(height)
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

func visibleRoomResultRows(height int) int {
	if height <= 0 {
		return 14
	}
	return maxInt(5, height-11)
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

func (m *roomScreenModel) moveRoomResultPage(delta int) {
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

func (m *roomScreenModel) moveRoomResultCursor(delta int) {
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

type roomScreenModel struct {
	roomDayCursor       int
	roomDaysSelected    map[int]bool
	roomPeriodCursor    int
	roomPeriodsSelected map[int]bool
	roomResults         []app.RoomAvailableResult
	roomResultPage      int
	roomResultCursor    int
}
type roomScreenService interface {
	RoomAvailabilityForDays(context.Context, app.RoomAvailabilityForDaysOptions) ([]app.RoomAvailableResult, error)
}

func (m *roomScreenModel) Start() {
	*m = roomScreenModel{roomDaysSelected: map[int]bool{}, roomPeriodsSelected: map[int]bool{}}
}
func (m *roomScreenModel) Loaded(results []app.RoomAvailableResult) {
	m.roomResults = results
	m.roomResultPage = 0
	m.roomResultCursor = 0
}
func (m *roomScreenModel) Update(msg tea.Msg, route screen, loading bool, ctx context.Context, service roomScreenService) (childAction, bool) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return childAction{}, false
	}
	k := key.String()
	if k == "ctrl+c" || keyMatches(k, "q", "ㅂ") {
		return childAction{cmd: tea.Quit}, true
	}
	if k == "esc" || keyMatches(k, "b", "ㅠ") {
		target := screenHome
		if route == screenRoomPeriod {
			target = screenRoomDay
		}
		if route == screenRoomResult {
			target = screenRoomPeriod
		}
		return childAction{navigate: true, target: target, setError: true}, true
	}
	if route == screenRoomResult {
		if keyMatches(k, "r", "ㄱ") {
			return childAction{setLoading: true, loading: true, setError: true, cmd: m.Load(ctx, service, true)}, true
		}
		if loading {
			return childAction{}, false
		}
		switch {
		case k == "up" || keyMatches(k, "k", "ㅏ"):
			m.moveRoomResultCursor(-1)
		case k == "down" || keyMatches(k, "j", "ㅓ"):
			m.moveRoomResultCursor(1)
		case k == "left":
			m.moveRoomResultPage(-1)
		case k == "right":
			m.moveRoomResultPage(1)
		default:
			return childAction{}, false
		}
		return childAction{}, true
	}
	if route == screenRoomDay {
		switch {
		case k == "up" || keyMatches(k, "k", "ㅏ"):
			m.roomDayCursor = maxInt(0, m.roomDayCursor-1)
		case k == "down" || keyMatches(k, "j", "ㅓ"):
			m.roomDayCursor = minInt(len(roomDayOptions)-1, m.roomDayCursor+1)
		case k == " ":
			m.toggleRoomDayCurrent()
		case keyMatches(k, "a", "ㅁ"):
			m.toggleRoomAllDays()
		case k == "enter":
			if len(m.selectedRoomDays()) == 0 {
				return childAction{setError: true, err: errors.New("요일을 하나 이상 선택하세요")}, true
			}
			m.roomPeriodCursor = 0
			if m.roomPeriodsSelected == nil {
				m.roomPeriodsSelected = map[int]bool{}
			}
			return childAction{navigate: true, target: screenRoomPeriod, setError: true}, true
		}
		return childAction{}, true
	}
	switch {
	case k == "up" || keyMatches(k, "k", "ㅏ"):
		m.roomPeriodCursor = maxInt(0, m.roomPeriodCursor-1)
	case k == "down" || keyMatches(k, "j", "ㅓ"):
		m.roomPeriodCursor = minInt(7, m.roomPeriodCursor+1)
	case k == " ":
		m.toggleRoomPeriodCurrent()
	case keyMatches(k, "a", "ㅁ"):
		m.toggleRoomAllPeriods()
	case k == "enter":
		if len(m.selectedRoomPeriods()) == 0 {
			return childAction{setError: true, err: errors.New("교시를 하나 이상 선택하세요")}, true
		}
		m.Loaded(nil)
		return childAction{navigate: true, target: screenRoomResult, setError: true, cmd: m.Load(ctx, service, false)}, true
	}
	return childAction{}, true
}
func (m roomScreenModel) View(width, height int, route screen, err error) string {
	switch route {
	case screenRoomDay:
		return m.renderRoomDayView(width, err)
	case screenRoomPeriod:
		return m.renderRoomPeriodView(width, err)
	default:
		return m.renderRoomResultPanel(width, height)
	}
}
