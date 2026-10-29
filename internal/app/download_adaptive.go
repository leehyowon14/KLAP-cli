package app

import (
	"context"
	"math"
	"sync"
	"time"
)

type downloadGate struct {
	mu              sync.Mutex
	maximum, target int
	active          []string
	adaptive        bool
	last            time.Time
	bytes           int64
	seen, window    map[string]int64
}

func newDownloadGate(maximum int, adaptive bool) *downloadGate {
	target := maximum
	if adaptive {
		target = min(3, maximum)
	}
	return &downloadGate{maximum: maximum, target: target, adaptive: adaptive, last: time.Now(), seen: map[string]int64{}, window: map[string]int64{}}
}
func (g *downloadGate) acquire(ctx context.Context, id string) bool {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return false
		}
		g.mu.Lock()
		if len(g.active) < g.target {
			g.active = append(g.active, id)
			g.mu.Unlock()
			return true
		}
		g.mu.Unlock()
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
		}
	}
}
func (g *downloadGate) release(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, v := range g.active {
		if v == id {
			g.active = append(g.active[:i], g.active[i+1:]...)
			return
		}
	}
}
func (g *downloadGate) paused(id string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, v := range g.active {
		if v == id {
			return i >= g.target
		}
	}
	return false
}
func (g *downloadGate) wait(ctx context.Context, id string, changed func(bool)) {
	if !g.paused(id) {
		return
	}
	changed(true)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for g.paused(id) {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
	changed(false)
}
func adaptiveDownloadLimit(current, maximum int, rate float64) int {
	if rate >= float64(current)*10_000_000*0.85 {
		return min(maximum, current+1)
	}
	estimated := max(1, int(math.Ceil(rate/10_000_000)))
	if estimated < current && rate < float64(current-1)*10_000_000*0.65 {
		return max(1, current-1)
	}
	return current
}
func (g *downloadGate) observe(p LectureDownloadProgress, now time.Time) {
	if !g.adaptive || p.Stage != LectureStageDownload {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	previous, ok := g.seen[p.Lecture.ID]
	if ok && p.Bytes > previous {
		delta := p.Bytes - previous
		g.bytes += delta
		g.window[p.Lecture.ID] += delta
	}
	g.seen[p.Lecture.ID] = p.Bytes
	elapsed := now.Sub(g.last).Seconds()
	if elapsed < 5 {
		return
	}
	next := adaptiveDownloadLimit(g.target, g.maximum, float64(g.bytes)/elapsed)
	if next > g.target {
		allFast := len(g.active) >= g.target
		for i, id := range g.active {
			if i < g.target && float64(g.window[id])/elapsed < 8_500_000 {
				allFast = false
			}
		}
		if !allFast {
			next = g.target
		}
	}
	g.target = next
	g.bytes = 0
	g.window = map[string]int64{}
	g.last = now
}
