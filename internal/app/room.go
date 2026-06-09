package app

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kw-klap/klap-cli/internal/cache"
	"github.com/kw-klap/klap-cli/internal/klas"
)

const roomIndexConcurrency = 8

var weekdayLabels = map[int]string{
	1: "월",
	2: "화",
	3: "수",
	4: "목",
	5: "금",
}

var buildingAliases = []struct {
	aliases   []string
	canonical string
}{
	{aliases: []string{"화", "화도", "화도관"}, canonical: "화도관"},
	{aliases: []string{"한천", "한천재"}, canonical: "한천재"},
	{aliases: []string{"한울", "한울관"}, canonical: "한울관"},
	{aliases: []string{"참", "참빛", "참빛관"}, canonical: "참빛관"},
	{aliases: []string{"옥", "옥의", "옥의관"}, canonical: "옥의관"},
	{aliases: []string{"연", "연구", "연구관"}, canonical: "연구관"},
	{aliases: []string{"새빛", "새", "새빛관"}, canonical: "새빛관"},
	{aliases: []string{"비", "비마", "비마관"}, canonical: "비마관"},
	{aliases: []string{"누", "누리", "누리관"}, canonical: "누리관"},
	{aliases: []string{"기", "기념", "기념관"}, canonical: "기념관"},
}

type RoomSlot struct {
	Weekday int
	Period  int
}

type RoomBusyRow struct {
	Room       string
	Weekday    int
	Period     int
	Span       int
	CourseName string
	CourseCode string
	Professor  string
	SubjectID  string
}

type RoomIndex struct {
	Term        string
	GeneratedAt time.Time
	Rooms       map[string]RoomSchedule
	Warnings    []RoomIndexWarning
}

type RoomSchedule struct {
	Room string
	Busy []RoomBusySlot
}

type RoomBusySlot struct {
	Weekday     int
	Period      int
	Span        int
	SubjectName string
	Professor   string
	CourseCode  string
	SubjectID   string
}

type RoomIndexWarning struct {
	SubjectID   string
	SubjectName string
	Error       string
}

type RoomIndexRoom struct {
	Room      string
	BusyCount int
}

type RoomIndexResult struct {
	TermValue      string
	Rooms          []RoomIndexRoom
	Warnings       []string
	Cached         bool
	CacheCreatedAt time.Time
}

type RoomQueryResult struct {
	TermValue      string
	Room           string
	Candidates     []string
	FreeSlots      []RoomSlot
	BusyRows       []RoomBusyRow
	Warnings       []string
	Cached         bool
	CacheCreatedAt time.Time
}

type RoomAvailableRoom struct {
	Room string
}

type RoomAvailableResult struct {
	TermValue      string
	Weekday        int
	Periods        []int
	Rooms          []RoomAvailableRoom
	Warnings       []string
	Cached         bool
	CacheCreatedAt time.Time
}

type RoomIndexOptions struct {
	TermValue  string
	Refresh    bool
	Building   string
	OnProgress func(done int, total int, label string)
	User       UserOption
}

type RoomQueryOptions struct {
	Room       string
	TermValue  string
	Refresh    bool
	Building   string
	OnProgress func(done int, total int, label string)
	User       UserOption
}

type RoomAvailableOptions struct {
	TermValue  string
	Refresh    bool
	Day        string
	Duration   string
	Periods    []int
	Building   string
	OnProgress func(done int, total int, label string)
	User       UserOption
}

func NormalizeRoom(value string) string {
	compactRaw := strings.Join(strings.Fields(strings.TrimSpace(value)), "")
	compactFolded := strings.ToLower(compactRaw)
	if compactFolded == "" {
		return ""
	}
	for _, item := range buildingAliases {
		aliasTokens := make([]string, 0, len(item.aliases)+1)
		seen := make(map[string]struct{}, len(item.aliases)+1)
		for _, alias := range append([]string{item.canonical}, item.aliases...) {
			aliasToken := normalizeRoomToken(alias)
			if aliasToken == "" {
				continue
			}
			if _, ok := seen[aliasToken]; ok {
				continue
			}
			seen[aliasToken] = struct{}{}
			aliasTokens = append(aliasTokens, aliasToken)
		}
		sort.Slice(aliasTokens, func(i, j int) bool { return len(aliasTokens[i]) > len(aliasTokens[j]) })
		for _, aliasToken := range aliasTokens {
			if strings.HasPrefix(compactFolded, aliasToken) {
				suffix := strings.TrimSpace(compactRaw[len(aliasToken):])
				if suffix == "" {
					return item.canonical
				}
				return item.canonical + suffix
			}
		}
	}
	return compactRaw
}

func NormalizeBuilding(value string) string {
	token := normalizeRoomToken(value)
	for _, item := range buildingAliases {
		for _, alias := range item.aliases {
			if token == normalizeRoomToken(alias) {
				return item.canonical
			}
		}
	}
	return strings.TrimSpace(value)
}

func (s *Service) RoomIndex(ctx context.Context, opts RoomIndexOptions) (RoomIndexResult, error) {
	index, hit, cached, err := s.roomIndex(ctx, opts)
	if err != nil {
		return RoomIndexResult{}, err
	}
	rooms := make([]RoomIndexRoom, 0, len(index.Rooms))
	building := NormalizeBuilding(opts.Building)
	for room, schedule := range index.Rooms {
		if building != "" && !strings.HasPrefix(room, building) {
			continue
		}
		rooms = append(rooms, RoomIndexRoom{Room: room, BusyCount: busyPeriodCount(schedule.Busy)})
	}
	sort.Slice(rooms, func(i, j int) bool { return rooms[i].Room < rooms[j].Room })
	return RoomIndexResult{
		TermValue:      index.Term,
		Rooms:          rooms,
		Warnings:       roomWarningMessages(index.Warnings),
		Cached:         cached,
		CacheCreatedAt: hit.CreatedAt,
	}, nil
}

func (s *Service) RoomFree(ctx context.Context, opts RoomQueryOptions) (RoomQueryResult, error) {
	result, err := s.roomQuery(ctx, opts)
	if err != nil || result.Room == "" || len(result.Candidates) != 0 {
		return result, err
	}
	busy := make(map[string]struct{}, len(result.BusyRows))
	for _, row := range result.BusyRows {
		if row.Weekday >= 1 && row.Weekday <= 5 && row.Period >= 1 && row.Period <= 8 {
			for offset := 0; offset < normalizedSpan(row.Span); offset++ {
				busy[roomSlotKey(row.Weekday, row.Period+offset)] = struct{}{}
			}
		}
	}
	for weekday := 1; weekday <= 5; weekday++ {
		for period := 1; period <= 8; period++ {
			if _, ok := busy[roomSlotKey(weekday, period)]; ok {
				continue
			}
			result.FreeSlots = append(result.FreeSlots, RoomSlot{Weekday: weekday, Period: period})
		}
	}
	return result, nil
}

func (s *Service) RoomBusy(ctx context.Context, opts RoomQueryOptions) (RoomQueryResult, error) {
	return s.roomQuery(ctx, opts)
}

func (s *Service) RoomAvailable(ctx context.Context, opts RoomAvailableOptions) (RoomAvailableResult, error) {
	weekday := parseRoomWeekday(opts.Day)
	if weekday == 0 {
		return RoomAvailableResult{}, errors.New("--day에는 월, 월요일, mon, monday 같은 요일이 필요합니다")
	}
	periods := normalizeRoomPeriods(opts.Periods)
	if len(periods) == 0 {
		var err error
		periods, err = parseRoomDuration(opts.Duration)
		if err != nil {
			return RoomAvailableResult{}, err
		}
	}
	index, hit, cached, err := s.roomIndex(ctx, RoomIndexOptions{
		TermValue:  opts.TermValue,
		Refresh:    opts.Refresh,
		Building:   opts.Building,
		OnProgress: opts.OnProgress,
		User:       opts.User,
	})
	if err != nil {
		return RoomAvailableResult{}, err
	}
	return RoomAvailableResult{
		TermValue:      index.Term,
		Weekday:        weekday,
		Periods:        periods,
		Rooms:          availableRooms(index, weekday, periods, opts.Building),
		Warnings:       roomWarningMessages(index.Warnings),
		Cached:         cached,
		CacheCreatedAt: hit.CreatedAt,
	}, nil
}

func (s *Service) ClearRoomCache() (CacheClearResult, error) {
	removed, err := s.cacheStore.ClearPrefix("room-index:")
	if err != nil {
		return CacheClearResult{}, err
	}
	return CacheClearResult{Removed: removed}, nil
}

func (s *Service) roomQuery(ctx context.Context, opts RoomQueryOptions) (RoomQueryResult, error) {
	if strings.TrimSpace(opts.Room) == "" {
		return RoomQueryResult{}, errors.New("강의실명이 필요합니다")
	}
	index, hit, cached, err := s.roomIndex(ctx, RoomIndexOptions{
		TermValue:  opts.TermValue,
		Refresh:    opts.Refresh,
		Building:   opts.Building,
		OnProgress: opts.OnProgress,
		User:       opts.User,
	})
	if err != nil {
		return RoomQueryResult{}, err
	}
	candidates := matchRooms(index.Rooms, opts.Room, opts.Building)
	result := RoomQueryResult{
		TermValue:      index.Term,
		Warnings:       roomWarningMessages(index.Warnings),
		Cached:         cached,
		CacheCreatedAt: hit.CreatedAt,
	}
	if len(candidates) != 1 {
		result.Candidates = candidates
		return result, nil
	}
	result.Room = candidates[0]
	result.BusyRows = roomBusyRows(index.Rooms[result.Room])
	sortRoomBusyRows(result.BusyRows)
	return result, nil
}

func (s *Service) roomIndex(ctx context.Context, opts RoomIndexOptions) (RoomIndex, cache.Hit, bool, error) {
	studentID, err := s.selectedStudentID(ctx, opts.User)
	if err != nil {
		return RoomIndex{}, cache.Hit{}, false, err
	}
	client, err := s.authenticatedClient(ctx, studentID)
	if err != nil {
		return RoomIndex{}, cache.Hit{}, false, err
	}
	term, client, err := s.termForSyllabus(ctx, studentID, client, opts.TermValue)
	if err != nil {
		return RoomIndex{}, cache.Hit{}, false, err
	}

	key := roomIndexCacheKey(term.Value)
	if !opts.Refresh {
		var cached RoomIndex
		if hit, ok, cacheErr := s.cacheStore.Get(key, &cached); cacheErr == nil && ok {
			return cached, hit, true, nil
		}
	}

	index, err := buildRoomIndexFromKlas(ctx, client, term.Value, opts.OnProgress)
	if err != nil {
		return RoomIndex{}, cache.Hit{}, false, err
	}
	_ = s.cacheStore.Set(key, roomIndexCacheTTL(), index)
	return index, cache.Hit{}, false, nil
}

func buildRoomIndexFromKlas(ctx context.Context, client *klas.Client, termValue string, onProgress func(done int, total int, label string)) (RoomIndex, error) {
	items, err := client.SyllabusList(ctx, termValue, "", "")
	if err != nil {
		return RoomIndex{}, err
	}
	index := RoomIndex{
		Term:        termValue,
		Rooms:       make(map[string]RoomSchedule),
		GeneratedAt: time.Now(),
	}
	type job struct {
		item      klas.SyllabusListItem
		subjectID string
	}
	jobs := make(chan job)
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0
	total := len(items)

	worker := func() {
		defer wg.Done()
		for current := range jobs {
			times, err := client.SyllabusTimeInfo(ctx, current.subjectID)
			mu.Lock()
			if err != nil {
				index.Warnings = append(index.Warnings, RoomIndexWarning{
					SubjectID:   current.subjectID,
					SubjectName: current.item.KoreanName,
					Error:       err.Error(),
				})
			} else {
				addRoomIndexTimes(&index, current.item, current.subjectID, times)
			}
			done++
			if onProgress != nil {
				onProgress(done, total, firstNonEmpty(current.item.KoreanName, current.subjectID))
			}
			mu.Unlock()
		}
	}

	concurrency := roomIndexConcurrency
	if total < concurrency {
		concurrency = total
	}
	if concurrency <= 0 {
		return index, nil
	}
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}
	for _, item := range items {
		subjectID, err := item.SubjectID()
		if err != nil {
			mu.Lock()
			index.Warnings = append(index.Warnings, RoomIndexWarning{
				SubjectID:   item.CourseCode(),
				SubjectName: item.KoreanName,
				Error:       err.Error(),
			})
			done++
			if onProgress != nil {
				onProgress(done, total, firstNonEmpty(item.KoreanName, item.CourseCode()))
			}
			mu.Unlock()
			continue
		}
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return index, ctx.Err()
		case jobs <- job{item: item, subjectID: subjectID}:
		}
	}
	close(jobs)
	wg.Wait()
	sortRoomWarnings(index.Warnings)
	for room, schedule := range index.Rooms {
		sortRoomBusySlots(schedule.Busy)
		index.Rooms[room] = schedule
	}
	return index, nil
}

func addRoomIndexTimes(index *RoomIndex, item klas.SyllabusListItem, subjectID string, times []klas.SyllabusTime) {
	for _, timeInfo := range times {
		room := NormalizeRoom(timeInfo.Room)
		if room == "" {
			continue
		}
		weekday := parseRoomWeekday(timeInfo.Weekday)
		if weekday == 0 {
			continue
		}
		schedule := index.Rooms[room]
		if schedule.Room == "" {
			schedule.Room = room
		}
		for _, span := range roomPeriodSpans(timeInfo.Periods) {
			schedule.Busy = append(schedule.Busy, RoomBusySlot{
				Weekday:     weekday,
				Period:      span.Period,
				Span:        span.Span,
				SubjectName: item.KoreanName,
				CourseCode:  item.CourseCode(),
				Professor:   item.Professor,
				SubjectID:   subjectID,
			})
		}
		index.Rooms[room] = schedule
	}
}

func matchRooms(rooms map[string]RoomSchedule, query string, buildingFilter string) []string {
	normalizedQuery := NormalizeRoom(query)
	building := NormalizeBuilding(buildingFilter)
	if normalizedQuery == "" {
		return nil
	}
	if schedule, ok := rooms[normalizedQuery]; ok && len(schedule.Busy) > 0 {
		if building == "" || strings.HasPrefix(normalizedQuery, building) {
			return []string{normalizedQuery}
		}
		return nil
	}
	queryToken := normalizeRoomToken(normalizedQuery)
	matches := make([]string, 0)
	for room := range rooms {
		if building != "" && !strings.HasPrefix(room, building) {
			continue
		}
		roomToken := normalizeRoomToken(room)
		if strings.Contains(roomToken, queryToken) || strings.Contains(queryToken, roomToken) {
			matches = append(matches, room)
		}
	}
	sort.Strings(matches)
	return matches
}

func parseRoomWeekday(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "월", "월요일":
		return 1
	case "화", "화요일":
		return 2
	case "수", "수요일":
		return 3
	case "목", "목요일":
		return 4
	case "금", "금요일":
		return 5
	case "토", "토요일":
		return 6
	case "mon", "monday":
		return 1
	case "tue", "tues", "tuesday":
		return 2
	case "wed", "wednesday":
		return 3
	case "thu", "thur", "thurs", "thursday":
		return 4
	case "fri", "friday":
		return 5
	case "sat", "saturday":
		return 6
	default:
		return 0
	}
}

func parseRoomDuration(value string) ([]int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("--duration에는 1-3 또는 5 같은 교시 범위가 필요합니다")
	}
	parts := strings.Split(value, "-")
	if len(parts) > 2 {
		return nil, errors.New("--duration은 1-3 또는 5 형식이어야 합니다")
	}
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || start < 1 || start > 8 {
		return nil, errors.New("--duration 교시는 1부터 8 사이여야 합니다")
	}
	end := start
	if len(parts) == 2 {
		end, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || end < 1 || end > 8 || end < start {
			return nil, errors.New("--duration 범위는 1-8 사이의 오름차순이어야 합니다")
		}
	}
	periods := make([]int, 0, end-start+1)
	for period := start; period <= end; period++ {
		periods = append(periods, period)
	}
	return periods, nil
}

func availableRooms(index RoomIndex, weekday int, periods []int, buildingFilter string) []RoomAvailableRoom {
	building := NormalizeBuilding(buildingFilter)
	rooms := make([]RoomAvailableRoom, 0, len(index.Rooms))
	for room, schedule := range index.Rooms {
		if building != "" && !strings.HasPrefix(room, building) {
			continue
		}
		busy := roomBusySet(schedule.Busy)
		available := true
		for _, period := range periods {
			if _, ok := busy[roomSlotKey(weekday, period)]; ok {
				available = false
				break
			}
		}
		if available {
			rooms = append(rooms, RoomAvailableRoom{Room: room})
		}
	}
	sort.Slice(rooms, func(i, j int) bool { return rooms[i].Room < rooms[j].Room })
	return rooms
}

func normalizeRoomPeriods(periods []int) []int {
	values := make([]int, 0, len(periods))
	seen := make(map[int]struct{}, len(periods))
	for _, period := range periods {
		if period < 1 || period > 8 {
			continue
		}
		if _, ok := seen[period]; ok {
			continue
		}
		seen[period] = struct{}{}
		values = append(values, period)
	}
	sort.Ints(values)
	return values
}

func roomBusyRows(schedule RoomSchedule) []RoomBusyRow {
	rows := make([]RoomBusyRow, 0, len(schedule.Busy))
	for _, slot := range schedule.Busy {
		rows = append(rows, RoomBusyRow{
			Room:       schedule.Room,
			Weekday:    slot.Weekday,
			Period:     slot.Period,
			Span:       normalizedSpan(slot.Span),
			CourseName: slot.SubjectName,
			CourseCode: slot.CourseCode,
			Professor:  slot.Professor,
			SubjectID:  slot.SubjectID,
		})
	}
	return rows
}

func roomBusySet(slots []RoomBusySlot) map[string]struct{} {
	busy := make(map[string]struct{}, len(slots))
	for _, slot := range slots {
		if slot.Weekday <= 0 || slot.Period <= 0 {
			continue
		}
		for offset := 0; offset < normalizedSpan(slot.Span); offset++ {
			busy[roomSlotKey(slot.Weekday, slot.Period+offset)] = struct{}{}
		}
	}
	return busy
}

func busyPeriodCount(slots []RoomBusySlot) int {
	count := 0
	for _, slot := range slots {
		count += normalizedSpan(slot.Span)
	}
	return count
}

type roomPeriodSpan struct {
	Period int
	Span   int
}

func roomPeriodSpans(periods []int) []roomPeriodSpan {
	values := make([]int, 0, len(periods))
	seen := make(map[int]struct{}, len(periods))
	for _, period := range periods {
		if period <= 0 {
			continue
		}
		if _, ok := seen[period]; ok {
			continue
		}
		seen[period] = struct{}{}
		values = append(values, period)
	}
	sort.Ints(values)
	spans := make([]roomPeriodSpan, 0, len(values))
	for index := 0; index < len(values); {
		start := values[index]
		end := start
		index++
		for index < len(values) && values[index] == end+1 {
			end = values[index]
			index++
		}
		spans = append(spans, roomPeriodSpan{Period: start, Span: end - start + 1})
	}
	return spans
}

func normalizedSpan(span int) int {
	if span <= 0 {
		return 1
	}
	return span
}

func sortRoomBusyRows(rows []RoomBusyRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Weekday != rows[j].Weekday {
			return rows[i].Weekday < rows[j].Weekday
		}
		if rows[i].Period != rows[j].Period {
			return rows[i].Period < rows[j].Period
		}
		if rows[i].Room != rows[j].Room {
			return rows[i].Room < rows[j].Room
		}
		return rows[i].CourseName < rows[j].CourseName
	})
}

func sortRoomBusySlots(slots []RoomBusySlot) {
	sort.Slice(slots, func(i, j int) bool {
		if slots[i].Weekday != slots[j].Weekday {
			return slots[i].Weekday < slots[j].Weekday
		}
		if slots[i].Period != slots[j].Period {
			return slots[i].Period < slots[j].Period
		}
		return slots[i].SubjectName < slots[j].SubjectName
	})
}

func sortRoomWarnings(warnings []RoomIndexWarning) {
	sort.Slice(warnings, func(i, j int) bool {
		if warnings[i].SubjectName != warnings[j].SubjectName {
			return warnings[i].SubjectName < warnings[j].SubjectName
		}
		return warnings[i].SubjectID < warnings[j].SubjectID
	})
}

func roomWarningMessages(warnings []RoomIndexWarning) []string {
	messages := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		label := firstNonEmpty(warning.SubjectName, warning.SubjectID)
		if label == "" {
			label = "unknown"
		}
		if strings.TrimSpace(warning.Error) == "" {
			messages = append(messages, label)
			continue
		}
		messages = append(messages, label+": "+warning.Error)
	}
	return messages
}

func roomSlotKey(weekday int, period int) string {
	return strconv.Itoa(weekday) + ":" + strconv.Itoa(period)
}

func roomIndexCacheKey(termValue string) string {
	return "room-index:" + strings.TrimSpace(termValue)
}

func roomIndexCacheTTL() time.Duration {
	return 24 * time.Hour * 30 * 4
}

func normalizeRoomToken(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), ""))
}

func RoomWeekdayLabel(weekday int) string {
	if label, ok := weekdayLabels[weekday]; ok {
		return label
	}
	return strconv.Itoa(weekday)
}
