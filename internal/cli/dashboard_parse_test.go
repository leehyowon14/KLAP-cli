package cli

import (
	"testing"
)

func TestDashboardRefreshFlag(t *testing.T) {
	if !dashboardRefreshFlag([]string{"--refresh"}) {
		t.Fatal("dashboardRefreshFlag() expected true")
	}
	if unknown := firstUnknownDashboardArg([]string{"--refresh", "--user", "20260000"}); unknown != "" {
		t.Fatalf("firstUnknownDashboardArg() = %q", unknown)
	}
	if unknown := firstUnknownDashboardArg([]string{"--bad"}); unknown != "--bad" {
		t.Fatalf("firstUnknownDashboardArg() unknown = %q", unknown)
	}
}
