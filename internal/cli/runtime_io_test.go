package cli

import (
	"bytes"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"testing"
	"time"
)

func TestRunnerOpenUsesInjectedOutputAndOpener(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	wantErr := errors.New("open failed")
	calls := 0
	r := Runner{Out: &out, Opener: func(target, kind string) error {
		calls++
		if target != "https://example.test" || kind != "URL" {
			t.Fatal("opener arguments changed")
		}
		return wantErr
	}}
	if err := r.openAndPrintURL("https://example.test", nil); !errors.Is(err, wantErr) {
		t.Fatalf("error=%v", err)
	}
	if calls != 1 || out.String() != "URL: https://example.test\n" {
		t.Fatalf("calls=%d output=%q", calls, out.String())
	}
	if err := r.openAndPrintURL("ignored", wantErr); !errors.Is(err, wantErr) || calls != 1 {
		t.Fatalf("error=%v calls=%d", err, calls)
	}
}

func TestRunnerClockControlsLectureStatus(t *testing.T) {
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	for _, tc := range []struct {
		now  time.Time
		want string
	}{{start.Add(-time.Second), "예정"}, {start, "미완료"}, {end.Add(time.Second), "기간 종료"}} {
		r := Runner{Clock: func() time.Time { return tc.now }}
		if got := r.lectureStatusLabel(klas.Lecture{StartAt: &start, EndAt: &end}, 0); got != tc.want {
			t.Fatalf("status=%q want=%q", got, tc.want)
		}
	}
}

func TestRunnerBufferDisablesInPlaceProgress(t *testing.T) {
	var out bytes.Buffer
	if (Runner{Out: &out}).stdoutSupportsInPlaceProgress() {
		t.Fatal("buffer treated as terminal")
	}
}
