package app

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/klas"
	"strings"
)

type BoardOptions struct {
	User                                          UserOption
	Kind, TermValue, SubjectID, BoardNo, MasterNo string
}
type BoardDetailResult struct {
	Detail     NoticeDetail
	Files      []klas.BoardAttachment
	FilesError string
}
type AttachmentDownloadResult struct{ Path string }

func boardRequest[T any](ctx context.Context, s *Service, o BoardOptions, request func(*klas.Client, Term, Course) (T, error)) (T, error) {
	var zero T
	if o.Kind != "notice" && o.Kind != "material" {
		return zero, errors.New("지원하지 않는 게시판입니다")
	}
	if strings.TrimSpace(o.TermValue) == "" || strings.TrimSpace(o.SubjectID) == "" {
		return zero, errors.New("학기와 과목 정보가 필요합니다")
	}
	id, e := s.selectedStudentID(ctx, o.User)
	if e != nil {
		return zero, e
	}
	c, e := s.authenticatedClient(ctx, id)
	if e != nil {
		return zero, e
	}
	term, c, e := s.termForSyllabus(ctx, id, c, o.TermValue)
	if e != nil {
		return zero, e
	}
	var course *Course
	for i := range term.Courses {
		if term.Courses[i].Value == o.SubjectID {
			course = &term.Courses[i]
			break
		}
	}
	if course == nil {
		return zero, errors.New("수강 과목을 찾을 수 없습니다")
	}
	return executeSessionRequest(ctx, s, id, &c, func(c *klas.Client) (T, error) { return request(c, term, *course) })
}
func (s *Service) BoardList(ctx context.Context, o BoardOptions) ([]Notice, error) {
	return boardRequest(ctx, s, o, func(c *klas.Client, t Term, course Course) ([]Notice, error) {
		rows, e := c.BoardPosts(ctx, o.Kind, t.Value, course)
		if e != nil {
			return nil, e
		}
		result := make([]Notice, 0, len(rows))
		for _, r := range rows {
			result = append(result, noticeModel(r))
		}
		return result, nil
	})
}
func (s *Service) BoardDetail(ctx context.Context, o BoardOptions) (BoardDetailResult, error) {
	return boardRequest(ctx, s, o, func(c *klas.Client, t Term, course Course) (BoardDetailResult, error) {
		d, e := c.BoardPost(ctx, o.Kind, t.Value, course, o.BoardNo, o.MasterNo)
		if e != nil {
			return BoardDetailResult{}, e
		}
		r := BoardDetailResult{Detail: noticeDetailModel(d)}
		r.Files, e = c.BoardAttachments(ctx, d.Attachment)
		if errors.Is(e, klas.ErrSessionExpired) {
			return r, e
		}
		if e != nil {
			r.FilesError = e.Error()
		}
		return r, nil
	})
}
func (s *Service) BoardDownload(ctx context.Context, o BoardOptions, fileSN, directory string) (AttachmentDownloadResult, error) {
	return boardRequest(ctx, s, o, func(c *klas.Client, t Term, course Course) (AttachmentDownloadResult, error) {
		d, e := c.BoardPost(ctx, o.Kind, t.Value, course, o.BoardNo, o.MasterNo)
		if e != nil {
			return AttachmentDownloadResult{}, e
		}
		files, e := c.BoardAttachments(ctx, d.Attachment)
		if e != nil {
			return AttachmentDownloadResult{}, e
		}
		for _, f := range files {
			if f.FileSN == fileSN {
				path, e := c.DownloadBoardAttachment(ctx, f, directory)
				return AttachmentDownloadResult{path}, e
			}
		}
		return AttachmentDownloadResult{}, errors.New("첨부파일이 삭제되었거나 변경되었습니다. 게시물을 다시 불러와 주세요")
	})
}

func (s *Service) BoardRead(ctx context.Context, o BoardOptions, fileSN string) ([]byte, error) {
	return boardRequest(ctx, s, o, func(c *klas.Client, t Term, course Course) ([]byte, error) {
		d, e := c.BoardPost(ctx, o.Kind, t.Value, course, o.BoardNo, o.MasterNo)
		if e != nil {
			return nil, e
		}
		files, e := c.BoardAttachments(ctx, d.Attachment)
		if e != nil {
			return nil, e
		}
		for _, f := range files {
			if f.FileSN == fileSN {
				data, e := c.ReadBoardAttachment(ctx, f)
				return data, e
			}
		}
		return nil, errors.New("첨부파일이 삭제되었거나 변경되었습니다. 게시물을 다시 불러와 주세요")
	})
}
