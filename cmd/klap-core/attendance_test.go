package main

import (
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"testing"
)

func TestAttendanceRows(t *testing.T) {
	if rows := attendanceRows(nil); rows == nil || len(rows) != 0 {
		t.Fatal("empty results must encode as an array")
	}
	rows := attendanceRows([]app.AttendanceRow{{Index: 1, Err: errors.New("조회 실패")}, {Index: 2, Sessions: []app.AttendanceSession{{Week: "1"}}}})
	if rows[0].Error != "조회 실패" || rows[1].Error != "" || rows[1].Sessions[0].Week != "1" {
		t.Fatal("partial failures must preserve successful courses and error text")
	}
}
