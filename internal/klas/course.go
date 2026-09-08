package klas

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type Term struct {
	Label   string   `json:"label"`
	Value   string   `json:"value"`
	Courses []Course `json:"subjList"`
}

type Course struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (c *Client) Courses(ctx context.Context) ([]Term, error) {
	body, err := c.do(ctx, http.MethodPost, "/std/cmn/frame/YearhakgiAtnlcSbjectList.do", map[string]any{})
	if err != nil {
		return nil, err
	}

	var terms []Term
	if err := decodeResponseJSON(body, &terms); err != nil {
		return nil, fmt.Errorf("수업 목록 응답 파싱 실패: %w", err)
	}

	for termIndex := range terms {
		terms[termIndex].Label = normalizeTermLabel(terms[termIndex].Label, terms[termIndex].Value)
		filtered := terms[termIndex].Courses[:0]
		for _, course := range terms[termIndex].Courses {
			if strings.TrimSpace(course.Name) == "" || strings.TrimSpace(course.Value) == "" {
				continue
			}
			filtered = append(filtered, course)
		}
		terms[termIndex].Courses = filtered
	}

	return terms, nil
}

func (c *Client) SetCourseContext(ctx context.Context, yearHakgi string, course Course) error {
	_, err := c.do(ctx, http.MethodPost, "/std/lis/evltn/LctrumHomeStdInfo.do", map[string]any{
		"selectYearhakgi": yearHakgi,
		"selectSubj":      course.Value,
		"selectChangeYn":  "Y",
	})
	if err != nil {
		return fmt.Errorf("과목 컨텍스트 변경 실패: %w", err)
	}
	return nil
}

func splitYearHakgi(yearHakgi string) (string, string) {
	parts := strings.FieldsFunc(strings.TrimSpace(yearHakgi), func(r rune) bool {
		return r == ',' || r == '-'
	})
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	if len(parts) == 1 && len(parts[0]) >= 5 {
		return parts[0][:4], parts[0][4:]
	}
	return yearHakgi, ""
}

func normalizeTermLabel(label string, value string) string {
	label = strings.TrimSpace(label)
	year, hakgi := splitYearHakgi(value)
	if year == "" || hakgi == "" {
		return label
	}

	seasonLabel := semesterLabel(hakgi)
	if seasonLabel == "" {
		if label != "" {
			return label
		}
		return fmt.Sprintf("%s년도 %s학기", year, hakgi)
	}
	return fmt.Sprintf("%s년도 %s", year, seasonLabel)
}

func semesterLabel(hakgi string) string {
	switch strings.TrimSpace(hakgi) {
	case "3":
		return "여름학기"
	case "4":
		return "겨울학기"
	default:
		return ""
	}
}
