package klas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type EvaluationTerm struct {
	Year        string
	Hakgi       string
	Value       string
	Label       string
	JudgeChasu  string
	JudgeName   string
	FromDate    string
	ToDate      string
	TermEnabled bool
}

type EvaluationCourse struct {
	Name            string
	Professor       string
	CourseType      string
	EngineeringType string
	Credits         string
	Evaluated       bool
	Engineering     bool
	ThisYear        string
	Hakgi           string
	OpenMajorCode   string
	OpenGrade       string
	OpenGwamokNo    string
	BunbanNo        string
	Raw             evaluationCourseItem `json:"-"`
}

type EvaluationForm struct {
	Questions       map[string]any
	BunbanOptions   map[string]any
	SmallCourse     string
	EngineeringView string
	Engineering     map[string]any
}

type EvaluationAnswerOptions struct {
	Choice             string
	Text               string
	DiscriminationOpt  string
	IncludeEngineering bool
}

type EvaluationSubmitResult struct {
	Raw string
}

type evaluationTermItem struct {
	ThisYear   string `json:"thisYear"`
	Hakgi      string `json:"hakgi"`
	JudgeChasu string `json:"judgeChasu"`
	FromDate   string `json:"fromDate"`
	ToDate     string `json:"toDate"`
}

type evaluationStudentItem struct {
	StudentID string `json:"hakbun"`
	Name      string `json:"kname"`
	Division  string `json:"gubun"`
	Grade     any    `json:"grade"`
	Status    string `json:"hakjukStat"`
	Major     string `json:"codeName1"`
}

type evaluationCourseItem struct {
	ThisYear        string         `json:"thisYear"`
	Hakgi           string         `json:"hakgi"`
	OpenMajorCode   string         `json:"openMajorCode"`
	OpenGrade       string         `json:"openGrade"`
	OpenGwamokNo    string         `json:"openGwamokNo"`
	BunbanNo        string         `json:"bunbanNo"`
	Name            string         `json:"gwamokKname"`
	Professor       string         `json:"memberName"`
	Credits         flexibleString `json:"hakjumNum"`
	CourseType      string         `json:"isuGubun"`
	EngineeringType string         `json:"engIsuGubun"`
	EngOpt          string         `json:"engOpt"`
	JudgeOpt        string         `json:"judgeOpt"`
}

func (c *Client) EvaluationTerm(ctx context.Context) (EvaluationTerm, error) {
	body, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/LctreEvlTermCheck.do", nil)
	if err != nil {
		return EvaluationTerm{}, err
	}

	item, enabled, err := parseEvaluationTerm(body)
	if err != nil {
		return EvaluationTerm{}, err
	}
	if !enabled {
		return EvaluationTerm{TermEnabled: false}, nil
	}
	term := buildEvaluationTerm(item)
	term.TermEnabled = true
	return term, nil
}

func (c *Client) EvaluationStudent(ctx context.Context) (evaluationStudentItem, error) {
	body, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/LctreEvlGetHakjuk.do", nil)
	if err != nil {
		return evaluationStudentItem{}, err
	}
	var item evaluationStudentItem
	if err := json.Unmarshal(body, &item); err != nil {
		return evaluationStudentItem{}, fmt.Errorf("수업평가 학적 응답 파싱 실패: %w", err)
	}
	return item, nil
}

func (c *Client) EvaluationCourses(ctx context.Context, term EvaluationTerm) ([]EvaluationCourse, error) {
	payload := evaluationTermPayload(term)
	body, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/LctreEvlsugangList.do", payload)
	if err != nil {
		return nil, err
	}

	var response []evaluationCourseItem
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("수업평가 과목 응답 파싱 실패: %w", err)
	}
	courses := make([]EvaluationCourse, 0, len(response))
	for _, item := range response {
		course := buildEvaluationCourse(item)
		if strings.TrimSpace(course.Name) == "" {
			continue
		}
		courses = append(courses, course)
	}
	return courses, nil
}

func (c *Client) EvaluationForm(ctx context.Context, term EvaluationTerm, course EvaluationCourse) (EvaluationForm, error) {
	payload := evaluationCoursePayload(term, course)
	if err := c.checkEvaluationTarget(ctx, payload); err != nil {
		return EvaluationForm{}, err
	}

	questions, err := c.evaluationMap(ctx, "/std/cps/inqire/LctreEvlGetque.do", payload, false)
	if err != nil {
		return EvaluationForm{}, fmt.Errorf("수업평가 문항 응답 파싱 실패: %w", err)
	}
	bunban, err := c.evaluationMap(ctx, "/std/cps/inqire/LctreEvlBunbanCheck.do", payload, true)
	if err != nil {
		return EvaluationForm{}, fmt.Errorf("수업평가 분반 문항 응답 파싱 실패: %w", err)
	}
	smallCourse, err := c.evaluationString(ctx, "/std/cps/inqire/LctreEvlBunbanSmallCheck.do", payload, true)
	if err != nil {
		return EvaluationForm{}, fmt.Errorf("수업평가 소규모 강좌 응답 파싱 실패: %w", err)
	}
	engineering, err := c.evaluationMap(ctx, "/std/cps/inqire/LctreEvlEngQuestion.do", payload, true)
	if err != nil {
		return EvaluationForm{}, fmt.Errorf("공학인증 문항 응답 파싱 실패: %w", err)
	}

	engineeringView := "0"
	if course.Engineering && len(engineering) > 0 {
		engineeringView = "1"
	}
	return EvaluationForm{
		Questions:       questions,
		BunbanOptions:   bunban,
		SmallCourse:     smallCourse,
		EngineeringView: engineeringView,
		Engineering:     engineering,
	}, nil
}

func (c *Client) SubmitEvaluation(ctx context.Context, term EvaluationTerm, course EvaluationCourse, form EvaluationForm, opts EvaluationAnswerOptions) (EvaluationSubmitResult, error) {
	payload := BuildEvaluationPayload(term, course, form, opts)
	body, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/insertEvl.do", payload)
	if err != nil {
		return EvaluationSubmitResult{}, err
	}
	if shouldSubmitEngineering(term, course, form, opts) {
		_, err = c.do(ctx, http.MethodPost, "/std/cps/inqire/insertEng.do", payload)
		if err != nil {
			return EvaluationSubmitResult{}, err
		}
	}
	return EvaluationSubmitResult{Raw: strings.TrimSpace(string(body))}, nil
}

func BuildEvaluationPayload(term EvaluationTerm, course EvaluationCourse, form EvaluationForm, opts EvaluationAnswerOptions) map[string]any {
	choice := firstNonEmpty(opts.Choice, "5")
	text := firstNonEmpty(opts.Text, "많은 도움 되었습니다. 한학기동안 감사했습니다.")
	discriminationOpt := firstNonEmpty(opts.DiscriminationOpt, "N")

	payload := evaluationCoursePayload(term, course)
	payload["list"] = form.Questions
	payload["info"] = map[string]any{}
	payload["size"] = len(form.Questions)
	payload["bunbanCheckInfo"] = form.BunbanOptions
	payload["smallOpt"] = form.SmallCourse
	payload["eng100Opt"] = mapString(form.BunbanOptions, "eng100Opt")
	payload["engOpt"] = boolYN(course.Engineering)
	payload["engQuestion"] = form.Engineering
	payload["engView"] = form.EngineeringView

	for i := 1; i <= 40; i++ {
		field := fmt.Sprintf("a%02d", i)
		payload[field] = ""
		payload[field+"reason"] = ""
		question := fmt.Sprintf("q%02d", i)
		if hasQuestion(form.Questions, question) {
			payload[field] = choice
		}
	}
	for i := 1; i <= 21; i++ {
		field := fmt.Sprintf("sa%02d", i)
		payload[field] = ""
		if i <= 16 {
			payload[field+"reason"] = ""
		}
	}
	payload["fa01"] = ""
	payload["fa01reason"] = ""
	payload["ta01"] = ""
	payload["ta01reason"] = ""
	payload["ta02"] = ""
	payload["ta02reason"] = ""
	payload["chamgo"] = ""
	payload["chamgo2"] = ""
	payload["chamgo2Opt"] = discriminationOpt
	payload["chamgo3"] = ""

	for _, group := range evaluationSpecialQuestionGroups() {
		if !strings.EqualFold(mapString(form.BunbanOptions, group.option), "Y") {
			continue
		}
		for _, index := range group.indices {
			question := fmt.Sprintf("sq%02d", index)
			if hasQuestion(form.Questions, question) {
				payload[fmt.Sprintf("sa%02d", index)] = choice
			}
		}
	}
	if strings.EqualFold(mapString(form.BunbanOptions, "eng100Opt"), "Y") && hasQuestion(form.Questions, "fq01") {
		payload["fa01"] = choice
	}
	if hasQuestion(form.Questions, "tq01") {
		payload["ta01"] = choice
	}
	if hasQuestion(form.Questions, "tq02") {
		payload["ta02"] = choice
	}
	if hasQuestion(form.Questions, "chamgo") || hasQuestion(form.Questions, "chamgoEval") {
		payload["chamgo"] = text
	}
	if hasQuestion(form.Questions, "chamgo3") || strings.EqualFold(mapString(form.BunbanOptions, "recOpt"), "Y") {
		payload["chamgo3"] = text
	}

	for i := 1; i <= 20; i++ {
		payload[fmt.Sprintf("ea%d", i)] = ""
		payload[fmt.Sprintf("ea2%d", i)] = ""
		payload[fmt.Sprintf("ea3%d", i)] = ""
	}
	payload["ea41"] = ""
	payload["chamgoEng"] = ""
	if shouldSubmitEngineering(term, course, form, opts) {
		for i := 1; i <= 20; i++ {
			if hasQuestion(form.Engineering, fmt.Sprintf("level%d", i)) {
				payload[fmt.Sprintf("ea%d", i)] = choice
			}
			if hasQuestion(form.Engineering, fmt.Sprintf("studyResult%d", i)) {
				payload[fmt.Sprintf("ea2%d", i)] = choice
				payload[fmt.Sprintf("ea3%d", i)] = choice
			}
		}
		payload["ea41"] = choice
		payload["chamgoEng"] = text
	}

	return payload
}

func (c *Client) checkEvaluationTarget(ctx context.Context, payload map[string]any) error {
	body, err := c.do(ctx, http.MethodPost, "/std/cps/inqire/lctreEvlCheck.do", payload)
	if err != nil {
		return err
	}
	var response []map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("수업평가 대상자 응답 파싱 실패: %w", err)
	}
	if len(response) == 0 {
		return errors.New("수업평가 대상자가 아닙니다")
	}
	return nil
}

func (c *Client) evaluationMap(ctx context.Context, path string, payload map[string]any, allowEmpty bool) (map[string]any, error) {
	body, err := c.do(ctx, http.MethodPost, path, payload)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(body)) == "" {
		if allowEmpty {
			return map[string]any{}, nil
		}
		return nil, errors.New("빈 응답입니다")
	}
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) evaluationString(ctx context.Context, path string, payload map[string]any, allowEmpty bool) (string, error) {
	body, err := c.do(ctx, http.MethodPost, path, payload)
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		if allowEmpty {
			return "", nil
		}
		return "", errors.New("빈 응답입니다")
	}
	var text string
	if err := json.Unmarshal(body, &text); err == nil {
		return strings.TrimSpace(text), nil
	}
	return strings.Trim(trimmed, `"`), nil
}

func parseEvaluationTerm(body []byte) (evaluationTermItem, bool, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed == `""` || trimmed == "null" || trimmed == "[]" {
		return evaluationTermItem{}, false, nil
	}

	var item evaluationTermItem
	if err := json.Unmarshal(body, &item); err == nil && strings.TrimSpace(item.ThisYear+item.Hakgi+item.JudgeChasu) != "" {
		return item, true, nil
	}

	var items []evaluationTermItem
	if err := json.Unmarshal(body, &items); err != nil {
		return evaluationTermItem{}, false, fmt.Errorf("수업평가 기간 응답 파싱 실패: %w", err)
	}
	switch len(items) {
	case 0:
		return evaluationTermItem{}, false, nil
	case 1:
		return items[0], true, nil
	default:
		return evaluationTermItem{}, false, errors.New("수업평가 기간이 중복으로 내려왔습니다")
	}
}

func buildEvaluationTerm(item evaluationTermItem) EvaluationTerm {
	year := strings.TrimSpace(item.ThisYear)
	hakgi := strings.TrimSpace(item.Hakgi)
	value := ""
	if year != "" && hakgi != "" {
		value = year + "," + hakgi
	}
	judgeChasu := strings.TrimSpace(item.JudgeChasu)
	return EvaluationTerm{
		Year:       year,
		Hakgi:      hakgi,
		Value:      value,
		Label:      rankTermLabel(year, hakgi),
		JudgeChasu: judgeChasu,
		JudgeName:  evaluationJudgeName(judgeChasu),
		FromDate:   strings.TrimSpace(item.FromDate),
		ToDate:     strings.TrimSpace(item.ToDate),
	}
}

func evaluationJudgeName(judgeChasu string) string {
	switch strings.TrimSpace(judgeChasu) {
	case "middle":
		return "중간평가"
	case "last":
		return "기말평가"
	default:
		return strings.TrimSpace(judgeChasu)
	}
}

func buildEvaluationCourse(item evaluationCourseItem) EvaluationCourse {
	return EvaluationCourse{
		Name:            strings.TrimSpace(item.Name),
		Professor:       strings.TrimSpace(item.Professor),
		CourseType:      strings.TrimSpace(item.CourseType),
		EngineeringType: strings.TrimSpace(item.EngineeringType),
		Credits:         item.Credits.String(),
		Evaluated:       strings.EqualFold(strings.TrimSpace(item.JudgeOpt), "Y"),
		Engineering:     strings.EqualFold(strings.TrimSpace(item.EngOpt), "Y"),
		ThisYear:        strings.TrimSpace(item.ThisYear),
		Hakgi:           strings.TrimSpace(item.Hakgi),
		OpenMajorCode:   strings.TrimSpace(item.OpenMajorCode),
		OpenGrade:       strings.TrimSpace(item.OpenGrade),
		OpenGwamokNo:    strings.TrimSpace(item.OpenGwamokNo),
		BunbanNo:        strings.TrimSpace(item.BunbanNo),
		Raw:             item,
	}
}

func evaluationTermPayload(term EvaluationTerm) map[string]any {
	return map[string]any{
		"thisYear":       term.Year,
		"hakgi":          term.Hakgi,
		"judgeChasu":     term.JudgeChasu,
		"judgeChasuName": term.JudgeName,
		"termYn":         "Y",
	}
}

func evaluationCoursePayload(term EvaluationTerm, course EvaluationCourse) map[string]any {
	payload := evaluationTermPayload(term)
	payload["thisYear"] = firstNonEmpty(course.ThisYear, term.Year)
	payload["hakgi"] = firstNonEmpty(course.Hakgi, term.Hakgi)
	payload["openGrade"] = course.OpenGrade
	payload["openGwamokNo"] = course.OpenGwamokNo
	payload["openMajorCode"] = course.OpenMajorCode
	payload["gwamokKname"] = course.Name
	payload["bunbanNo"] = course.BunbanNo
	payload["engOpt"] = boolYN(course.Engineering)
	return payload
}

type evaluationQuestionGroup struct {
	option  string
	indices []int
}

func evaluationSpecialQuestionGroups() []evaluationQuestionGroup {
	return []evaluationQuestionGroup{
		{option: "experimentOpt", indices: []int{1, 2}},
		{option: "englishOpt", indices: []int{3, 4}},
		{option: "tblOpt", indices: []int{7, 8}},
		{option: "pblOpt", indices: []int{9, 10}},
		{option: "discuss0onOpt", indices: []int{11, 12}},
		{option: "practiceOpt", indices: []int{13, 14}},
		{option: "etcOpt", indices: []int{15, 16}},
		{option: "recOpt", indices: []int{17, 18, 19, 20, 21}},
	}
}

func shouldSubmitEngineering(term EvaluationTerm, course EvaluationCourse, form EvaluationForm, opts EvaluationAnswerOptions) bool {
	return opts.IncludeEngineering &&
		course.Engineering &&
		strings.EqualFold(strings.TrimSpace(term.JudgeChasu), "last") &&
		strings.EqualFold(strings.TrimSpace(form.EngineeringView), "1")
}

func hasQuestion(values map[string]any, key string) bool {
	value, ok := values[key]
	if !ok {
		return false
	}
	return strings.TrimSpace(fmt.Sprint(value)) != "" && strings.TrimSpace(fmt.Sprint(value)) != "<nil>"
}

func boolYN(value bool) string {
	if value {
		return "Y"
	}
	return "N"
}
