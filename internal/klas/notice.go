package klas

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
	Raw        noticeItem `json:"-"`
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
	Raw         noticeDetailResponse `json:"-"`
}

type noticeListResponse struct {
	List []noticeItem    `json:"list"`
	Page *noticePageInfo `json:"page"`
}

type noticePageInfo struct {
	CurrentPage   flexibleString `json:"currentPage"`
	PageSize      flexibleString `json:"pageSize"`
	TotalElements flexibleString `json:"totalElements"`
	TotalPages    flexibleString `json:"totalPages"`
}

type noticeDetailResponse struct {
	Board *noticeItem `json:"board"`
}

type noticeItem struct {
	BoardNo    flexibleString `json:"boardNo"`
	MasterNo   flexibleString `json:"masterNo"`
	Title      string         `json:"title"`
	Content    string         `json:"content"`
	TopAt      string         `json:"topAt"`
	AtchFileID flexibleString `json:"atchFileId"`
	ReadCount  flexibleString `json:"readCnt"`
	UserName   string         `json:"userNm"`
	RegistDt   string         `json:"registDt"`
	FileCount  flexibleString `json:"fileCnt"`
}

func (c *Client) Notices(ctx context.Context, yearHakgi string, course Course) ([]Notice, error) {
	if err := c.SetCourseContext(ctx, yearHakgi, course); err != nil {
		return nil, err
	}

	allNotices := make([]Notice, 0)
	seen := make(map[string]struct{})
	for currentPage := 0; ; currentPage++ {
		body, err := c.do(ctx, http.MethodPost, "/std/lis/sport/d052b8f845784c639f036b102fdc3023/BoardStdList.do", map[string]any{
			"selectYearhakgi": yearHakgi,
			"selectSubj":      course.Value,
			"currentPage":     currentPage,
			"searchCondition": "ALL",
			"searchKeyword":   nil,
		})
		if err != nil {
			return nil, err
		}

		notices, totalPages, err := parseNoticeListResponse(body)
		if err != nil {
			return nil, err
		}
		for _, notice := range notices {
			key := notice.BoardNo + ":" + notice.MasterNo
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			allNotices = append(allNotices, notice)
		}
		if currentPage >= totalPages-1 {
			break
		}
	}
	return allNotices, nil
}

func parseNoticeListResponse(body []byte) ([]Notice, int, error) {
	var response noticeListResponse
	if err := decodeResponseJSON(body, &response); err != nil {
		return nil, 0, fmt.Errorf("공지 목록 응답 파싱 실패: %w", err)
	}

	notices := make([]Notice, 0, len(response.List))
	for _, item := range response.List {
		boardNo := item.BoardNo.String()
		masterNo := item.MasterNo.String()
		if strings.TrimSpace(boardNo) == "" || strings.TrimSpace(masterNo) == "" {
			continue
		}
		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = "제목 없음"
		}
		notices = append(notices, Notice{
			BoardNo:    boardNo,
			MasterNo:   masterNo,
			Title:      title,
			Author:     strings.TrimSpace(item.UserName),
			Registered: parseKlasDateTime(item.RegistDt),
			Top:        strings.EqualFold(item.TopAt, "Y"),
			ReadCount:  item.ReadCount.String(),
			FileCount:  item.FileCount.String(),
			Raw:        item,
		})
	}
	return notices, noticeTotalPages(response.Page), nil
}

func noticeTotalPages(page *noticePageInfo) int {
	if page == nil {
		return 1
	}
	totalPages, err := strconv.Atoi(strings.TrimSpace(page.TotalPages.String()))
	if err != nil || totalPages < 1 {
		return 1
	}
	return totalPages
}

func (c *Client) NoticeDetail(ctx context.Context, yearHakgi string, course Course, boardNo string, masterNo string) (NoticeDetail, error) {
	if err := c.SetCourseContext(ctx, yearHakgi, course); err != nil {
		return NoticeDetail{}, err
	}

	body, err := c.do(ctx, http.MethodPost, "/std/lis/sport/d052b8f845784c639f036b102fdc3023/BoardStdView.do", map[string]any{
		"selectYearhakgi": yearHakgi,
		"selectSubj":      course.Value,
		"boardNo":         strings.TrimSpace(boardNo),
		"masterNo":        strings.TrimSpace(masterNo),
		"cmd":             "select",
	})
	if err != nil {
		return NoticeDetail{}, err
	}

	var response noticeDetailResponse
	if err := decodeResponseJSON(body, &response); err != nil {
		return NoticeDetail{}, fmt.Errorf("공지 상세 응답 파싱 실패: %w", err)
	}
	if response.Board == nil {
		return NoticeDetail{}, schemaError(errors.New("공지 상세 응답에 board 본문이 없습니다"))
	}
	if strings.TrimSpace(response.Board.Title) == "" && strings.TrimSpace(response.Board.Content) == "" {
		return NoticeDetail{}, schemaError(errors.New("공지 상세 응답에 제목과 본문이 없습니다"))
	}

	board := response.Board
	return NoticeDetail{
		BoardNo:     firstNonEmpty(board.BoardNo.String(), boardNo),
		MasterNo:    firstNonEmpty(board.MasterNo.String(), masterNo),
		Title:       firstNonEmpty(strings.TrimSpace(board.Title), "제목 없음"),
		ContentText: htmlToText(board.Content),
		Author:      strings.TrimSpace(board.UserName),
		Registered:  parseKlasDateTime(board.RegistDt),
		Top:         strings.EqualFold(board.TopAt, "Y"),
		ReadCount:   board.ReadCount.String(),
		Attachment:  board.AtchFileID.String(),
		Raw:         response,
	}, nil
}
