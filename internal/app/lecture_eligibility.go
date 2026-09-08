package app

import (
	"errors"
	"time"
)

// ValidateLectureAttendance requires a known, current attendance window.
func ValidateLectureAttendance(row LectureRow, now time.Time) error {
	l := row.Lecture
	if row.ID == "" || lectureAttendKey(l) == "" {
		return errors.New("자동 수강을 지원하지 않는 강의입니다")
	}
	if l.StartAt == nil || l.EndAt == nil || l.EndAt.Before(*l.StartAt) {
		return errors.New("수강 가능 기간을 확인할 수 없습니다")
	}
	if now.Before(*l.StartAt) {
		return errors.New("아직 수강 기간이 시작되지 않았습니다")
	}
	if now.After(*l.EndAt) {
		return errors.New("수강 기간이 종료되었습니다")
	}
	if !lectureNeedsAttendance(l, now) {
		return errors.New("이미 수강을 완료한 강의입니다")
	}
	return nil
}
