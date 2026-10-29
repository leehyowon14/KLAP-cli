package app

import (
	"testing"
	"time"
)

func TestRecentNoticeWindowIsTimeBased(t *testing.T) {
	now := time.Now()
	w, e := noticeWindow(AgentQuery{Intent: "recent"}, now)
	if e != nil || now.Sub(w.From) != 7*24*time.Hour || !w.Until.Equal(now) {
		t.Fatalf("%+v %v", w, e)
	}
	for _, q := range []AgentQuery{{Intent: "in_period"}, {Intent: "overview", Period: &QueryPeriod{Name: "today"}}, {Intent: "recent", Period: &QueryPeriod{Name: "today"}}} {
		if _, e := noticeWindow(q, now); e == nil {
			t.Fatalf("accepted %+v", q)
		}
	}
}
