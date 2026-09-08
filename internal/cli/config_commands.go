package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"strconv"
	"strings"
)

func (r Runner) runConfig(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		settings, err := service.ConfigSettings()
		if err != nil {
			return err
		}
		r.printConfigSettings(settings)
		return nil
	}

	switch args[0] {
	case "list":
		settings, err := service.ConfigSettings()
		if err != nil {
			return err
		}
		r.printConfigSettings(settings)
		return nil
	case "set":
		if len(args) < 3 {
			return errors.New("usage: klap config set <key> <value>")
		}
		update, err := parseConfigUpdate(args[1], strings.Join(args[2:], " "))
		if err != nil {
			return err
		}
		settings, err := service.UpdateConfig(update)
		if err != nil {
			return err
		}
		r.printConfigSettings(settings)
		return nil
	case "reset":
		if len(args) != 1 {
			return errors.New("usage: klap config reset")
		}
		settings, err := service.ResetConfigSettings()
		if err != nil {
			return err
		}
		r.printConfigSettings(settings)
		return nil
	case "reminder":
		return r.runConfigReminder(ctx, service, args[1:])
	case "download":
		return r.runConfigDownload(ctx, service, args[1:])
	default:
		return fmt.Errorf("unknown config command: %s", args[0])
	}
}

func (r Runner) runConfigReminder(ctx context.Context, service *app.Service, args []string) error {
	_ = ctx

	if len(args) == 0 {
		settings, err := service.ReminderSettings()
		if err != nil {
			return err
		}
		r.printReminderSettings(settings)
		return nil
	}

	name, useExistingList, ok, err := parseReminderConfigArgs(args)
	if err != nil {
		return err
	}
	if ok {
		settings, err := service.SetReminderConfig(name, useExistingList)
		if err != nil {
			return err
		}
		r.printReminderSettings(settings)
		return nil
	}

	return errors.New(`usage: klap config reminder [--name "Kwangwoon Univ." [--use-existing-list]]`)
}

func (r Runner) runConfigDownload(ctx context.Context, service *app.Service, args []string) error {
	_ = ctx
	if len(args) == 0 {
		settings, err := service.DownloadSettings()
		if err != nil {
			return err
		}
		r.printDownloadSettings(settings)
		return nil
	}

	opts, ok, err := parseDownloadConfigArgs(args)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New(`usage: klap config download [--dir <다운로드 폴더>] [--concurrency <동시 다운로드 수>] [--caffeinate|--no-caffeinate] [--keep-partial|--no-keep-partial]`)
	}
	settings, err := service.SetDownloadConfig(opts.Dir, opts.Concurrency, opts.Caffeinate, opts.KeepPartial)
	if err != nil {
		return err
	}
	r.printDownloadSettings(settings)
	return nil
}

type downloadConfigArgs struct {
	Dir         string
	Concurrency int
	Caffeinate  *bool
	KeepPartial *bool
}

func parseDownloadConfigArgs(args []string) (downloadConfigArgs, bool, error) {
	var opts downloadConfigArgs
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 >= len(args) {
				return downloadConfigArgs{}, false, errors.New("--dir에는 다운로드 폴더 경로가 필요합니다")
			}
			opts.Dir = args[i+1]
			i++
		case "--concurrency":
			if i+1 >= len(args) {
				return downloadConfigArgs{}, false, errors.New("--concurrency에는 1 이상의 정수가 필요합니다")
			}
			concurrency, err := strconv.Atoi(args[i+1])
			if err != nil || concurrency <= 0 {
				return downloadConfigArgs{}, false, errors.New("--concurrency에는 1 이상의 정수가 필요합니다")
			}
			opts.Concurrency = concurrency
			i++
		case "--caffeinate":
			value := true
			opts.Caffeinate = &value
		case "--no-caffeinate":
			value := false
			opts.Caffeinate = &value
		case "--keep-partial":
			value := true
			opts.KeepPartial = &value
		case "--no-keep-partial":
			value := false
			opts.KeepPartial = &value
		default:
			return downloadConfigArgs{}, false, fmt.Errorf("unknown download config option: %s", args[i])
		}
	}
	if strings.TrimSpace(opts.Dir) == "" && opts.Concurrency <= 0 && opts.Caffeinate == nil && opts.KeepPartial == nil {
		return downloadConfigArgs{}, false, nil
	}
	return opts, true, nil
}

func parseReminderConfigArgs(args []string) (name string, useExistingList bool, ok bool, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			if i+1 >= len(args) {
				return "", false, false, errors.New("--name에는 리마인더 목록 이름이 필요합니다")
			}
			name = args[i+1]
			i++
		case "--use-existing-list":
			useExistingList = true
		default:
			return "", false, false, fmt.Errorf("unknown reminder config option: %s", args[i])
		}
	}
	if name == "" {
		return "", false, false, nil
	}
	return name, useExistingList, true, nil
}

func (r Runner) printReminderSettings(settings app.ReminderSettings) {
	_, _ = fmt.Fprintf(r.Out, "리마인더 목록: %s\n", settings.ListName)
	_, _ = fmt.Fprintf(r.Out, "기존 목록만 사용: %s\n", yesNo(settings.UseExistingList))
	_, _ = fmt.Fprintf(r.Out, "알림: 마감 %d분 전\n", settings.AlarmBeforeMin)
}

func (r Runner) printDownloadSettings(settings app.DownloadSettings) {
	_, _ = fmt.Fprintf(r.Out, "다운로드 폴더: %s\n", settings.Dir)
	_, _ = fmt.Fprintf(r.Out, "동시 다운로드: %d\n", settings.Concurrency)
	_, _ = fmt.Fprintf(r.Out, "절전 방지: %s\n", yesNo(settings.Caffeinate))
	_, _ = fmt.Fprintf(r.Out, "부분 파일 보존: %s\n", yesNo(settings.KeepPartial))
}

func (r Runner) printConfigSettings(settings app.ConfigSettings) {
	_, _ = fmt.Fprintln(r.Out, "설정")
	_, _ = fmt.Fprintf(r.Out, "term: %s\n", emptyFallback(settings.Term.Value, "자동"))
	_, _ = fmt.Fprintf(r.Out, "reminder.name: %s\n", settings.Reminder.ListName)
	_, _ = fmt.Fprintf(r.Out, "reminder.use-existing-list: %s\n", yesNo(settings.Reminder.UseExistingList))
	_, _ = fmt.Fprintf(r.Out, "reminder.alarm-before-min: %d\n", settings.Reminder.AlarmBeforeMin)
	_, _ = fmt.Fprintf(r.Out, "download.dir: %s\n", settings.Download.Dir)
	_, _ = fmt.Fprintf(r.Out, "download.concurrency: %d\n", settings.Download.Concurrency)
	_, _ = fmt.Fprintf(r.Out, "download.caffeinate: %s\n", yesNo(settings.Download.Caffeinate))
	_, _ = fmt.Fprintf(r.Out, "download.keep-partial: %s\n", yesNo(settings.Download.KeepPartial))
	_, _ = fmt.Fprintf(r.Out, "transcript.concurrency: %d\n", settings.Transcript.Concurrency)
}
