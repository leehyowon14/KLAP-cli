package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

type downloadRunStub struct {
	runCalls, cleanupCalls int
	cleanupErr             error
}

func (s *downloadRunStub) Run() (app.LectureDownloadPipelineResult, error) {
	s.runCalls++
	return app.LectureDownloadPipelineResult{}, context.Canceled
}
func (s *downloadRunStub) CancelAndCleanup() error { s.cleanupCalls++; return s.cleanupErr }

func TestDownloadCleanupRunsInCommandAndReportsErrors(t *testing.T) {
	wantErr := errors.New("cleanup failed")
	run := &downloadRunStub{cleanupErr: wantErr}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	child := lectureDownloadModel{ctx: ctx, cancel: cancel, pipeline: run, updates: make(chan tea.Msg, 1)}
	cmd := child.cancelAndCleanup()
	if !child.canceling || ctx.Err() == nil || run.cleanupCalls != 0 {
		t.Fatal("cleanup ran outside command or did not cancel")
	}
	message := cmd().(lectureDownloadCleanupMsg)
	if run.cleanupCalls != 1 || !errors.Is(message.err, wantErr) {
		t.Fatalf("message=%#v calls=%d", message, run.cleanupCalls)
	}
	root := model{active: screenDownloadProgress, download: downloadScreenModel{downloadProgress: &child}}
	next, _ := root.Update(message)
	got := next.(model)
	if !got.download.downloadProgress.done || !errors.Is(got.download.downloadProgress.err, wantErr) {
		t.Fatal("cleanup error hidden")
	}
	next, _ = got.Update(lectureDownloadDoneMsg{err: context.Canceled})
	got = next.(model)
	if !errors.Is(got.download.downloadProgress.err, wantErr) || !strings.Contains(got.download.downloadProgress.View(), wantErr.Error()) {
		t.Fatal("late terminal replaced or hid cleanup failure")
	}
}

func TestDownloadNavigationWaitsForRunCommand(t *testing.T) {
	for _, quit := range []bool{false, true} {
		run := &downloadRunStub{}
		ctx, cancel := context.WithCancel(context.Background())
		child := lectureDownloadModel{ctx: ctx, cancel: cancel, pipeline: run, updates: make(chan tea.Msg, 1)}
		cmd := child.cancelAndWait(quit)
		if run.runCalls != 0 || ctx.Err() == nil {
			t.Fatal("wait ran outside command")
		}
		message := cmd()
		if run.runCalls != 1 {
			t.Fatal("did not wait for run")
		}
		if quit {
			if _, ok := message.(tea.QuitMsg); !ok {
				t.Fatalf("message=%T", message)
			}
		} else {
			root := model{active: screenDownloadProgress, download: downloadScreenModel{downloadProgress: &child}}
			next, _ := root.Update(message)
			got := next.(model)
			if got.active != screenHome || got.download.downloadProgress != nil {
				t.Fatal("home transition missing")
			}
		}
	}
}

func TestDownloadCleanupIgnoresStaleNavigation(t *testing.T) {
	current := make(chan tea.Msg, 1)
	previous := make(chan tea.Msg, 1)
	root := model{active: screenDownloadProgress, download: downloadScreenModel{downloadProgress: &lectureDownloadModel{updates: current}}}
	next, _ := root.Update(lectureDownloadCleanupMsg{updates: previous})
	got := next.(model)
	if got.active != screenDownloadProgress || got.download.downloadProgress == nil || got.download.downloadProgress.updates != current {
		t.Fatal("stale cleanup navigated away")
	}
}

func TestDownloadCleanupCannotBeBypassedByNavigation(t *testing.T) {
	child := lectureDownloadModel{canceling: true, updates: make(chan tea.Msg, 1)}
	root := model{active: screenDownloadProgress, download: downloadScreenModel{downloadProgress: &child}}
	for _, key := range []tea.KeyMsg{{Type: tea.KeyEsc}, {Type: tea.KeyCtrlC}, {Type: tea.KeyRunes, Runes: []rune("h")}} {
		next, cmd := root.Update(key)
		got := next.(model)
		if cmd != nil || got.active != screenDownloadProgress || got.download.downloadProgress == nil {
			t.Fatal("navigation bypassed pending cleanup")
		}
	}
	next, _ := child.Update(lectureDownloadDoneMsg{})
	if next.(lectureDownloadModel).done {
		t.Fatal("terminal progress bypassed pending cleanup")
	}
}
