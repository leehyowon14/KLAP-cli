package klas

import (
	"context"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/kwcommons"
)

func (c *Client) ResolveNativeLectureContentID(ctx context.Context, l Lecture) (string, error) {
	form, err := lectureViewerForm(l)
	if err != nil {
		return "", err
	}
	body, err := c.doForm(ctx, "/spv/lis/lctre/viewer/LctreCntntsViewSpvPage.do", form, true)
	if err != nil {
		return "", err
	}
	return nativeViewerContentID(body)
}
func nativeViewerContentID(body []byte) (string, error) {
	id := kwcommons.ExtractKWCommonsContentID(string(body))
	if id == "" {
		return "", fmt.Errorf("강의 뷰어에서 다운로드 가능한 영상 주소를 찾지 못했습니다")
	}
	return id, nil
}
