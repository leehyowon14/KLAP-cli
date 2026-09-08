package cli

import (
	"context"
	"strings"
	"testing"
)

func TestParseDownloadConfigArgs(t *testing.T) {
	opts, ok, err := parseDownloadConfigArgs([]string{"--dir", "downloads/course", "--concurrency", "12", "--no-caffeinate", "--keep-partial"})
	if err != nil {
		t.Fatalf("parseDownloadConfigArgs() error = %v", err)
	}
	if !ok || opts.Dir != "downloads/course" || opts.Concurrency != 12 || opts.Caffeinate == nil || *opts.Caffeinate || opts.KeepPartial == nil || !*opts.KeepPartial {
		t.Fatalf("parseDownloadConfigArgs() = %+v, %v", opts, ok)
	}
	if _, _, err := parseDownloadConfigArgs([]string{"--concurrency", "0"}); err == nil {
		t.Fatal("parseDownloadConfigArgs() expected concurrency error")
	}
	if _, _, err := parseDownloadConfigArgs([]string{"--bad"}); err == nil {
		t.Fatal("parseDownloadConfigArgs() expected error")
	}
}

func TestParseReminderConfigArgs(t *testing.T) {
	name, useExistingList, ok, err := parseReminderConfigArgs([]string{"--name", "To-do", "--use-existing-list"})
	if err != nil {
		t.Fatalf("parseReminderConfigArgs() error = %v", err)
	}
	if !ok || name != "To-do" || !useExistingList {
		t.Fatalf("parseReminderConfigArgs() = %q, %v, %v", name, useExistingList, ok)
	}
}

func TestConfigResetRejectsExtraArgs(t *testing.T) {
	err := (Runner{}).normalized().runConfig(context.Background(), nil, []string{"reset", "download"})
	if err == nil || !strings.Contains(err.Error(), "klap config reset") {
		t.Fatalf("runConfig(reset extra) error = %v", err)
	}
}
