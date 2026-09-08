package app

import (
	"testing"
	"time"
)

func TestFutureTime(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.Local)
	past := now.Add(-time.Minute)
	same := now
	future := now.Add(time.Minute)

	if futureTime(nil, now) {
		t.Fatal("futureTime(nil) should be false")
	}
	if futureTime(&past, now) {
		t.Fatal("past deadline should not be synced")
	}
	if !futureTime(&same, now) || !futureTime(&future, now) {
		t.Fatal("current or future deadline should be synced")
	}
}
