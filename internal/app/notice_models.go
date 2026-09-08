package app

import (
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"time"
)

type Notice struct {
	BoardNo    string
	MasterNo   string
	Title      string
	Author     string
	Registered *time.Time
	Top        bool
	ReadCount  string
	FileCount  string
}

func noticeModel(value klas.Notice) Notice {
	return Notice{
		BoardNo:    value.BoardNo,
		MasterNo:   value.MasterNo,
		Title:      value.Title,
		Author:     value.Author,
		Registered: value.Registered,
		Top:        value.Top,
		ReadCount:  value.ReadCount,
		FileCount:  value.FileCount,
	}
}

type NoticeDetail struct {
	BoardNo     string
	MasterNo    string
	Title       string
	ContentText string
	Author      string
	Registered  *time.Time
	Top         bool
	ReadCount   string
	Attachment  string
}

func noticeDetailModel(value klas.NoticeDetail) NoticeDetail {
	return NoticeDetail{
		BoardNo:     value.BoardNo,
		MasterNo:    value.MasterNo,
		Title:       value.Title,
		ContentText: value.ContentText,
		Author:      value.Author,
		Registered:  value.Registered,
		Top:         value.Top,
		ReadCount:   value.ReadCount,
		Attachment:  value.Attachment,
	}
}
