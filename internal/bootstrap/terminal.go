package bootstrap

import (
	"io"
	"os"

	"github.com/leehyowon14/KLAP-cli/internal/cli"
)

func detectTerminalCapabilities(out io.Writer, getenv func(string) string) cli.TerminalCapabilities {
	terminal := false
	if file, ok := out.(*os.File); ok {
		if info, err := file.Stat(); err == nil {
			terminal = info.Mode()&os.ModeCharDevice != 0
		}
	}
	dumb := getenv("TERM") == "dumb"
	hyperlinks := !dumb && getenv("KLAP_NO_HYPERLINKS") == "" && (terminal || getenv("KLAP_FORCE_HYPERLINKS") != "")
	return cli.TerminalCapabilities{Hyperlinks: hyperlinks, InPlaceProgress: terminal && !dumb}
}
