package academic

import (
	httpclient "KLAP/http"
	utils "KLAP/util"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type SemesterSet struct {
	Items map[string]semester
}

type semesterListResponse []semester

type semester struct {
	Value   string   `json:"value"`
	Label   string   `json:"label"`
	Courses []course `json:"subjList"`
}

func Load(cookies []*http.Cookie) (*SemesterSet, error) {
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

	data := &SemesterSet{
		Items: make(map[string]semester),
	}
	for _, semester := range rawData {
		data.Items[strings.Join(
			strings.Split(semester.Value, ","),
			"-")] = semester
	}

	return data, nil
}

func (sem SemesterSet) GetSemesterList() []string {
	var data []string
	for _, semester := range sem.Items {
		data = append(
			data,
			strings.Join(
				strings.Split(semester.Value, ","),
				"-"),
		)
	}
	sort.Strings(data)
	return data
}

func (sem SemesterSet) GetCourses(semester string) (map[string]course, error) {
	value, ok := sem.Items[semester]
	if !ok {
		return nil, fmt.Errorf("해당하는 학기에 수강한 수업이 없습니다.")
	}

	data := make(map[string]course)

	for _, course := range value.Courses {
		data[course.Label] = course
	}

	return data, nil
}

// NOTE: User.Attend(User.GetCourses("2025-1")[과목 label]). 내부적으로는 course.GetLessons?
// NOTE: 근데 이렇게 하면 외부에서 User.GetCourses()[].GetLessons 할 수 있는 문제가...
