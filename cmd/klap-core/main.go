package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"github.com/leehyowon14/KLAP-cli/internal/bootstrap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type request struct {
	Concurrency       int
	Adaptive          bool
	Transcribe        bool
	Locale            string
	Year              string
	ExpectedAccount   string
	Kind              string
	SubjectID         string
	BoardNo           string
	MasterNo          string
	FileSN            string
	Directory         string
	TimetableName     string
	AcademicName      string
	ReminderName      string
	TimetableExisting bool
	AcademicExisting  bool
	ReminderExisting  bool
	Command           string
	Selector          string
	TermValue         string
	ID                string
	StudentID         string
	Password          string
	IDs               []string
	Decisions         map[string]app.SyncDecision
}
type event struct {
	Kind  string `json:"kind"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func emit(kind string, data any) {
	_ = json.NewEncoder(os.Stdout).Encode(event{Kind: kind, Data: data})
}
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var r request
	if err := json.NewDecoder(os.Stdin).Decode(&r); err != nil {
		fail(err)
		return
	}
	s, err := bootstrap.NewService()
	if err != nil {
		fail(err)
		return
	}
	if r.Command != "attend" && r.Command != "lecture-download" && r.Command != "lecture-transcribe" {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Minute)
		defer cancel()
	}
	switch r.Command {
	case "board-read":
		var data []byte
		data, err = s.BoardRead(ctx, app.BoardOptions{Kind: r.Kind, TermValue: r.TermValue, SubjectID: r.SubjectID, BoardNo: r.BoardNo, MasterNo: r.MasterNo}, r.FileSN)
		if err == nil {
			for offset := 0; offset < len(data); offset += 65536 {
				end := offset + 65536
				if end > len(data) {
					end = len(data)
				}
				emit("chunk", data[offset:end])
			}
			emit("result", map[string]int{"Bytes": len(data)})
		}
	case "board-list", "board-detail", "board-download":
		opts := app.BoardOptions{Kind: r.Kind, TermValue: r.TermValue, SubjectID: r.SubjectID, BoardNo: r.BoardNo, MasterNo: r.MasterNo}
		var value any
		switch r.Command {
		case "board-list":
			value, err = s.BoardList(ctx, opts)
		case "board-detail":
			value, err = s.BoardDetail(ctx, opts)
		case "board-download":
			value, err = s.BoardDownload(ctx, opts, r.FileSN, r.Directory)
		}
		if err == nil {
			emit("result", value)
		}
	case "syllabus":
		var v app.SyllabusResult
		v, err = s.Syllabus(ctx, app.SyllabusOptions{Selector: r.Selector, TermValue: r.TermValue})
		if err == nil {
			emit("result", v)
		}
	case "notice-detail":
		var v app.NoticeDetailResult
		v, err = s.NoticeDetail(ctx, r.ID, app.UserOption{})
		if err == nil {
			emit("result", v)
		}
	case "auth":
		err = s.Authenticate(ctx, r.StudentID, r.Password)
		if err == nil {
			emit("done", nil)
		}
	case "users":
		var v []app.UserRow
		v, err = s.Users(ctx)
		if err == nil {
			emit("result", v)
		}
	case "snapshot":
		users, accountErr := s.Users(ctx)
		if accountErr != nil {
			err = accountErr
			break
		}
		user, account, accountErr := notificationAccount(users)
		if accountErr != nil {
			err = accountErr
			break
		}
		result := map[string]any{"account": account}
		if name, nameErr := s.StudentName(ctx, user); nameErr == nil {
			result["studentName"] = name
		}
		problems := []string{}
		table, e := s.Timetable(ctx, app.TimetableOptions{User: user, Refresh: true})
		if e != nil {
			problems = append(problems, e.Error())
		} else {
			result["timetable"] = table
		}
		assignments, e := s.AssignmentList(ctx, app.AssignmentListOptions{User: user, Refresh: true})
		if e != nil {
			problems = append(problems, e.Error())
		} else {
			result["assignments"] = assignments
		}
		notices, e := s.NoticeList(ctx, app.NoticeListOptions{User: user, Refresh: true})
		if e != nil {
			problems = append(problems, e.Error())
		} else {
			result["notices"] = notices
		}
		lectures, e := s.LectureList(ctx, app.LectureListOptions{User: user, Refresh: true})
		if e != nil {
			problems = append(problems, e.Error())
		} else {
			rows := []map[string]any{}
			for _, l := range lectures {
				reason := ""
				if e := app.ValidateLectureAttendance(l, time.Now()); e != nil {
					reason = e.Error()
				}
				rows = append(rows, map[string]any{"row": l, "reason": reason})
			}
			result["lectures"] = rows
		}
		result["errors"] = problems
		emit("result", result)
	case "open":
		var v app.OpenURLResult
		v, err = s.LectureOpenURL(ctx, r.ID, app.UserOption{})
		if err == nil {
			emit("result", v)
		}
	default:
		if handler, ok := handlers[r.Command]; ok {
			err = handler(ctx, s, r)
		} else {
			err = fmt.Errorf("지원하지 않는 명령: %s", r.Command)
		}
	}
	if err != nil {
		fail(err)
	}
}
func fail(err error) {
	_ = json.NewEncoder(os.Stdout).Encode(event{Kind: "error", Error: err.Error()})
	os.Exit(1)
}

var handlers = map[string]func(context.Context, *app.Service, request) error{}
