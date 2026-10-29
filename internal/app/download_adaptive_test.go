package app

import (
	"context"
	"testing"
	"time"
)

func TestAdaptiveDownloadLimit(t *testing.T) {
	for _, v := range []struct {
		current, max int
		rate         float64
		want         int
	}{{2, 8, 20e6, 3}, {8, 8, 100e6, 8}, {2, 8, 1e6, 1}, {1, 8, 0, 1}, {3, 8, 22e6, 3}} {
		if got := adaptiveDownloadLimit(v.current, v.max, v.rate); got != v.want {
			t.Fatalf("%+v got %d", v, got)
		}
	}
	g := newDownloadGate(8, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if g.acquire(ctx, "cancel") {
		t.Fatal("cancelled admission")
	}
	g.observe(LectureDownloadProgress{Lecture: LectureRow{ID: "x"}, Stage: LectureStageDownload, Bytes: 1000}, time.Now().Add(6*time.Second))
	if g.bytes != 0 {
		t.Fatal("resume offset counted as throughput")
	}
}

func TestAdaptivePauseResume(t *testing.T) {
	g := newDownloadGate(8, true)
	if g.target != 3 {
		t.Fatal("must start with three")
	}
	for _, id := range []string{"a", "b", "c"} {
		if !g.acquire(context.Background(), id) {
			t.Fatal("admission")
		}
	}
	g.target = 2
	if !g.paused("c") || g.paused("a") {
		t.Fatal("pause priority")
	}
	g.release("a")
	if g.paused("c") {
		t.Fatal("must resume after slot freed")
	}
	small := newDownloadGate(1, true)
	if small.target != 1 {
		t.Fatal("maximum ignored")
	}
}

func TestAdaptiveRequiresEveryVideoFast(t *testing.T) {
	for _, slow := range []bool{false, true} {
		g := newDownloadGate(6, true)
		start := g.last
		for _, id := range []string{"a", "b", "c"} {
			g.acquire(context.Background(), id)
			g.observe(LectureDownloadProgress{Lecture: LectureRow{ID: id}, Stage: LectureStageDownload}, start)
		}
		for i, id := range []string{"a", "b", "c"} {
			bytes := int64(60_000_000)
			if slow && id == "b" {
				bytes = 1_000_000
			}
			now := start.Add(4 * time.Second)
			if i == 2 {
				now = start.Add(5 * time.Second)
			}
			g.observe(LectureDownloadProgress{Lecture: LectureRow{ID: id}, Stage: LectureStageDownload, Bytes: bytes}, now)
		}
		want := 4
		if slow {
			want = 3
		}
		if g.target != want {
			t.Fatalf("slow=%v target=%d", slow, g.target)
		}
	}
}
func TestPausedDownloadCancellation(t *testing.T) {
	g := newDownloadGate(3, true)
	g.acquire(context.Background(), "a")
	g.acquire(context.Background(), "b")
	g.target = 1
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() { g.wait(ctx, "b", func(bool) {}); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("paused worker did not cancel")
	}
}
