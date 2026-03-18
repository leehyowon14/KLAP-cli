package academic

import (
	httpclient "KLAP/http"
	utils "KLAP/util"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type AllSemester struct {
	Semesters map[string]semester
}

type semesterListResponse struct {
	Semesters []semester `json:"semester"`
}

type semester struct {
	Value    string    `json:"value"`
	Label    string    `json:"label"`
	Subjects []subject `json:"subjList"`
}

func Load(cookies []*http.Cookie) (*AllSemester, error) {
	req, _ := http.NewRequest(
		"POST",
		"https://klas.kw.ac.kr/std/cmn/frame/YearhakgiAtnlcSbjectList.do",
		bytes.NewBufferString("{}"),
	)
	utils.SetJsonRequestHeaders(req, cookies)

	res, err := httpclient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("과목 정보를 불러오는데 실패했습니다: %w", err)
	}
	defer res.Body.Close()

	var rawData semesterListResponse
	err = json.NewDecoder(res.Body).Decode(&rawData)
	if err != nil {
		return nil, fmt.Errorf("과목 정보를 해석하는데 실패했습니다: %w", err)
	}

	data := &AllSemester{
		Semesters: make(map[string]semester),
	}
	for _, semester := range rawData.Semesters {
		data.Semesters[strings.Join(
			strings.Split(semester.Value, ","),
			"-")] = semester
	}

	return data, nil
}
