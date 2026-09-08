package klas

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

func (c *Client) SyllabusEnrollment(ctx context.Context, term, subjectID, name string) (string, error) {
	year, semester := splitYearHakgi(term)
	if year == "" || semester == "" || strings.TrimSpace(subjectID) == "" {
		return "", errors.New("수강인원 조회 학기 또는 과목이 없습니다")
	}
	body, err := c.do(ctx, http.MethodPost, "/std/cps/atnlc/popup/LectrePlanStdCrtNum.do", map[string]any{
		"selectYear": year, "selecthakgi": semester, "selectYearHakgi": year + "," + semester,
		"selectSubj": subjectID, "gwamokName": name, "selectGrcode": "",
		"currentNum": "0", "numText": "", "randomNum": "", "stopFlag": "",
	})
	if err != nil {
		return "", err
	}
	var response struct {
		CurrentNum flexibleString `json:"currentNum"`
	}
	if err := decodeResponseJSON(body, &response); err != nil {
		return "", err
	}
	count, err := strconv.Atoi(strings.TrimSpace(response.CurrentNum.String()))
	if err != nil || count < 0 {
		return "", schemaError(errors.New("수강인원 응답에 유효한 인원이 없습니다"))
	}
	return strconv.Itoa(count), nil
}
