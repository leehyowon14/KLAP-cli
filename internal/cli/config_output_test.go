package cli

import (
	"bytes"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestRunnerConfigOutputMatchesBaseline(t *testing.T) {
	t.Setenv("KLAP_NO_HYPERLINKS", "1")
	var out bytes.Buffer
	r := (Runner{Out: &out}).normalized()
	r.printReminderSettings(app.ReminderSettings{})
	r.printDownloadSettings(app.DownloadSettings{})
	r.printConfigSettings(app.ConfigSettings{})
	want := "리마인더 목록: \n기존 목록만 사용: N\n알림: 마감 0분 전\n다운로드 폴더: \n동시 다운로드: 0\n절전 방지: N\n부분 파일 보존: N\n설정\nterm: 자동\nreminder.name: \nreminder.use-existing-list: N\nreminder.alarm-before-min: 0\ndownload.dir: \ndownload.concurrency: 0\ndownload.caffeinate: N\ndownload.keep-partial: N\ntranscript.concurrency: 0\n"
	if out.String() != want {
		t.Fatalf("output=%q want=%q", out.String(), want)
	}
}
