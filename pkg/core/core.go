// Package core is the supported Go entrypoint for KLAP clients. All KLAS
// authentication, identifiers, parsing, and use cases live in this repository.
// CLI, native apps, and MCP adapters must not reimplement those behaviors.
package core

import (
	"github.com/leehyowon14/KLAP-cli/internal/account"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"github.com/leehyowon14/KLAP-cli/internal/bootstrap"
	"github.com/leehyowon14/KLAP-cli/internal/domain"
)

const APIVersion = "1"

type (
	Service                = app.Service
	Credentials            = bootstrap.Credentials
	SavedUser              = account.User
	Session                = domain.Session
	UserOption             = app.UserOption
	UserRow                = app.UserRow
	Term                   = app.Term
	Course                 = app.Course
	CourseRef              = app.CourseRef
	CourseListOptions      = app.CourseListOptions
	TermListOptions        = app.TermListOptions
	AssignmentListOptions  = app.AssignmentListOptions
	AssignmentRow          = app.AssignmentRow
	AssignmentDetailResult = app.AssignmentDetailResult
	NoticeListOptions      = app.NoticeListOptions
	NoticeRow              = app.NoticeRow
	NoticeDetailResult     = app.NoticeDetailResult
	LectureListOptions     = app.LectureListOptions
	LectureRow             = app.LectureRow
	Lecture                = app.Lecture
	LectureAttendOptions   = app.LectureAttendOptions
	LectureAttendResult    = app.LectureAttendResult
	LectureProgress        = app.LectureProgress
	TimetableOptions       = app.TimetableOptions
	TimetableResult        = app.TimetableResult
	AcademicListOptions    = app.AcademicListOptions
	AcademicListResult     = app.AcademicListResult
	AttendanceListOptions  = app.AttendanceListOptions
	AttendanceListResult   = app.AttendanceListResult
	DueOptions             = app.DueOptions
	DueResult              = app.DueResult
	SearchOptions          = app.SearchOptions
	SearchResult           = app.SearchResult
	DashboardSyncOptions   = app.DashboardSyncOptions
	DashboardSyncResult    = app.DashboardSyncResult
	SyncDecision           = app.SyncDecision
	ConfigUpdate           = app.ConfigUpdate
)

var ValidateLectureAttendance = app.ValidateLectureAttendance
var NewCourseRef = app.NewCourseRef

// NewDesktop uses the existing OS Keychain and the configured KLAP directories.
func NewDesktop() (*Service, error) { return bootstrap.NewService() }

// NewWithCredentials uses the same application and adapters with a different
// credential store. Set KLAP_CONFIG_DIR/KLAP_CACHE_DIR once at process startup.
func NewWithCredentials(credentials Credentials) (*Service, error) {
	return bootstrap.NewServiceWithCredentials(credentials)
}
