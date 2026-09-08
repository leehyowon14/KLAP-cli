package cli

import (
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"os"
)

func (r Runner) linkifyForTerminal(text string) string {
	if !r.terminalHyperlinksEnabled() {
		return text
	}
	return hyperlinkURLs(text)
}

func (r Runner) openAndPrintURL(url string, err error) error {
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(r.Out, "URL: %s\n", r.linkifyForTerminal(url))
	return r.openExternal(url, "URL")
}

func (r Runner) terminalHyperlinksEnabled() bool {
	if os.Getenv("KLAP_NO_HYPERLINKS") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	if os.Getenv("KLAP_FORCE_HYPERLINKS") != "" {
		return true
	}

	return r.stdoutIsTerminal()
}

func (r Runner) stdoutSupportsInPlaceProgress() bool {
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return r.stdoutIsTerminal()
}

func (r Runner) stdoutIsTerminal() bool {
	file, ok := r.Out.(*os.File)
	if !ok {
		return false
	}
	stdout, err := file.Stat()
	if err != nil {
		return false
	}
	return stdout.Mode()&os.ModeCharDevice != 0
}

func (r Runner) lectureStatusLabel(lecture klas.Lecture, percent float64) string {
	if percent >= 100 {
		return "완료"
	}
	now := r.Clock()
	if lecture.StartAt != nil && now.Before(*lecture.StartAt) {
		return "예정"
	}
	if lecture.EndAt != nil && now.After(*lecture.EndAt) {
		return "기간 종료"
	}
	return "미완료"
}

func (r Runner) openExternal(target, kind string) error { return r.Opener(target, kind) }
