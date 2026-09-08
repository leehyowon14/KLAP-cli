package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type attendBatch struct {
	rows           []app.LectureRow
	selected       map[string]bool
	course, cursor int
	height         int
	queue          []app.LectureRow
	results        []string
	err            string
	done           bool
	stopping       bool
}

func (m *attendScreenModel) StartSelection(rows []app.LectureRow) {
	*m = attendScreenModel{batch: &attendBatch{rows: append([]app.LectureRow(nil), rows...), selected: map[string]bool{}}}
}

func (b *attendBatch) groups() []downloadCourseGroup {
	return (downloadScreenModel{downloadRows: b.rows}).downloadGroups()
}

func (b *attendBatch) currentRows() []app.LectureRow {
	groups := b.groups()
	if b.course < 0 || b.course >= len(groups) {
		return nil
	}
	return groups[b.course].rows
}

func (b *attendBatch) toggle(rows []app.LectureRow, now time.Time) {
	all := true
	for _, r := range rows {
		if app.ValidateLectureAttendance(r, now) == nil && !b.selected[r.ID] {
			all = false
		}
	}
	for _, r := range rows {
		if app.ValidateLectureAttendance(r, now) == nil {
			b.selected[r.ID] = !all
		} else {
			delete(b.selected, r.ID)
		}
	}
}

func (m *attendScreenModel) updateBatch(msg tea.Msg, route screen, ctx context.Context, service lectureAttender, now time.Time) childAction {
	b := m.batch
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		b.height = size.Height
	}
	if route == screenAttendProgress {
		if key, ok := msg.(tea.KeyMsg); ok {
			k := key.String()
			if k == "ctrl+c" || keyMatches(k, "q", "ㅂ") {
				if m.attendProgress != nil {
					m.attendProgress.cancel()
				}
				b.stopping = true
				return childAction{cmd: tea.Quit}
			}
			if b.done && (k == "esc" || keyMatches(k, "b", "ㅠ", "h", "ㅗ")) {
				*m = attendScreenModel{}
				return childAction{navigate: true, reload: true, refresh: true, target: screenLectures}
			}
			if k == "esc" || keyMatches(k, "h", "ㅗ") {
				b.stopping = true
				if m.attendProgress != nil {
					m.attendProgress.canceling = true
					m.attendProgress.cancel()
				}
				return childAction{}
			}
			if k == "up" {
				b.cursor = maxInt(0, b.cursor-1)
			}
			if k == "down" {
				b.cursor = minInt(maxInt(0, len(b.results)-1), b.cursor+1)
			}
		}
		if m.attendProgress == nil {
			return childAction{}
		}
		if done, ok := msg.(lectureAttendDoneMsg); ok && done.updates != m.attendProgress.updates {
			return childAction{}
		}
		updated, cmd := m.attendProgress.Update(msg)
		p := updated.(lectureAttendModel)
		m.attendProgress = &p
		if _, ok := msg.(lectureAttendDoneMsg); ok {
			status := "완료"
			if errors.Is(p.err, context.Canceled) {
				status = "중단"
			} else if p.err != nil {
				status = "실패: " + p.err.Error()
			} else if !p.progress.Completed {
				status = "미완료"
			}
			b.results = append(b.results, firstLectureLabel(b.queue[0])+" — "+status)
			b.queue = b.queue[1:]
			return m.nextBatch(ctx, service, now)
		}
		return childAction{cmd: cmd}
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return childAction{}
	}
	k := key.String()
	if k == "ctrl+c" || keyMatches(k, "q", "ㅂ") {
		return childAction{cmd: tea.Quit}
	}
	if k == "esc" || keyMatches(k, "b", "ㅠ", "n", "ㅜ") {
		if route == screenAttendConfirm {
			return childAction{navigate: true, routeOnly: true, target: screenAttendSelect}
		}
		*m = attendScreenModel{}
		return childAction{navigate: true, routeOnly: true, target: screenLectures}
	}
	if route == screenAttendConfirm {
		if k == "enter" || keyMatches(k, "y", "ㅛ") {
			b.results = nil
			b.done = false
			b.stopping = false
			b.cursor = 0
			return m.nextBatch(ctx, service, now)
		}
		return childAction{}
	}
	rows := b.currentRows()
	switch {
	case k == "left":
		b.course = maxInt(0, b.course-1)
		b.cursor = 0
	case k == "right":
		b.course = minInt(maxInt(0, len(b.groups())-1), b.course+1)
		b.cursor = 0
	case k == "up" || keyMatches(k, "k", "ㅏ"):
		b.cursor = (b.cursor + len(rows)) % (len(rows) + 1)
	case k == "down" || keyMatches(k, "j", "ㅓ"):
		b.cursor = (b.cursor + 1) % (len(rows) + 1)
	case k == " ":
		if b.cursor == 0 {
			b.toggle(rows, now)
		} else if b.cursor <= len(rows) {
			b.toggle(rows[b.cursor-1:b.cursor], now)
		}
	case keyMatches(k, "a", "ㅁ"):
		b.toggle(b.rows, now)
	case k == "enter":
		b.queue = nil
		seen := map[string]bool{}
		for _, r := range b.rows {
			if b.selected[r.ID] && !seen[r.ID] && app.ValidateLectureAttendance(r, now) == nil {
				b.queue = append(b.queue, r)
				seen[r.ID] = true
			}
		}
		if len(b.queue) == 0 {
			b.err = "수강 가능한 강의를 선택하세요"
			return childAction{}
		}
		b.err = ""
		return childAction{navigate: true, routeOnly: true, target: screenAttendConfirm}
	}
	return childAction{}
}

func (m *attendScreenModel) nextBatch(ctx context.Context, service lectureAttender, now time.Time) childAction {
	b := m.batch
	for len(b.queue) > 0 && !b.stopping && ctx.Err() == nil {
		r := b.queue[0]
		if err := app.ValidateLectureAttendance(r, now); err != nil {
			b.results = append(b.results, firstLectureLabel(r)+" — 제외: "+err.Error())
			b.queue = b.queue[1:]
			continue
		}
		runCtx, cancel := context.WithCancel(ctx)
		m.attendProgress = &lectureAttendModel{ctx: runCtx, cancel: cancel, service: service, row: r, updates: make(chan tea.Msg, 16), progress: initialAttendProgress(r), requireEligible: true}
		return childAction{navigate: true, routeOnly: true, target: screenAttendProgress, cmd: m.attendProgress.Init()}
	}
	for _, r := range b.queue {
		b.results = append(b.results, firstLectureLabel(r)+" — 중단")
	}
	b.queue = nil
	b.done = true
	m.attendProgress = nil
	return childAction{navigate: true, routeOnly: true, target: screenAttendProgress}
}

func (m attendScreenModel) batchView(width int, route screen) string {
	b := m.batch
	lines := []string{renderHeaderTitle(width, "Attend"), renderRule(width), ""}
	help := "←/→ 과목  ↑↓ 이동  space 선택  a 전체 과목  enter 다음  esc 뒤로  q 종료"
	if route == screenAttendSelect {
		groups := b.groups()
		rows := b.currentRows()
		if len(groups) > 0 {
			lines = append(lines, fmt.Sprintf("%d/%d  %s", b.course+1, len(groups), groups[b.course].name))
		}
		eligible, selected := 0, 0
		for _, r := range rows {
			if app.ValidateLectureAttendance(r, time.Now()) == nil {
				eligible++
				if b.selected[r.ID] {
					selected++
				}
			}
		}
		check := "[ ]"
		if eligible > 0 && selected == eligible {
			check = "[x]"
		}
		items := []string{fmt.Sprintf("%s 모두 선택 (%d/%d)", check, selected, eligible)}
		for _, r := range rows {
			check := "[ ]"
			if b.selected[r.ID] {
				check = "[x]"
			}
			reason := ""
			if err := app.ValidateLectureAttendance(r, time.Now()); err != nil {
				check = "[-]"
				reason = " · " + err.Error()
			}
			items = append(items, check+" "+firstLectureLabel(r)+reason)
		}
		if len(b.rows) == 0 {
			lines = append(lines, "온라인 강의가 없습니다")
		}
		limit := maxInt(3, b.height-11)
		if b.height == 0 {
			limit = 12
		}
		start := maxInt(0, b.cursor-limit/2)
		for i := start; i < len(items) && i < start+limit; i++ {
			marker := "  "
			if i == b.cursor {
				marker = "› "
			}
			lines = append(lines, marker+items[i])
		}
		lines = append(lines, b.err)
	} else if route == screenAttendConfirm {
		lines = append(lines, fmt.Sprintf("선택한 강의 %d개를 순서대로 수강할까요?", len(b.queue)), "시작 직전에 최신 수강 가능 기간을 다시 확인합니다.")
		help = "y/enter 시작  n/esc 선택으로  q 종료"
	} else {
		lines = append(lines, fmt.Sprintf("처리 %d개 · 남은 %d개", len(b.results), len(b.queue)))
		if m.attendProgress != nil {
			p := m.attendProgress
			lines = append(lines, p.row.CourseName+" · "+firstLectureLabel(p.row), renderLectureAttendProgress(p.progress, minInt(28, maxInt(1, width-20))))
		}
		if b.done {
			lines = append(lines, "선택 강의 처리 종료")
		} else if b.stopping {
			lines = append(lines, "수강 중단 중")
		}
		limit := maxInt(3, b.height-12)
		if b.height == 0 {
			limit = 10
		}
		start := b.cursor
		if !b.done {
			start = maxInt(0, len(b.results)-limit)
		}
		for i := start; i < len(b.results) && i < start+limit; i++ {
			lines = append(lines, b.results[i])
		}
		help = "esc 중단  q 종료"
		if b.done {
			help = "↑↓ 결과  b/esc 목록 새로고침  q 종료"
		}
	}
	for i := range lines {
		lines[i] = truncateText(lines[i], maxInt(1, width))
	}
	view := strings.Join(lines, "\n") + "\n\n" + renderHelpText(help, width)
	if route == screenAttendProgress {
		return appStyle.Render(view)
	}
	return view
}
