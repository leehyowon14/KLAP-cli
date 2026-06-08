# Lecture Evaluation Fixtures

KLAS `수업평가` 화면에서 확인한 API 흐름이다.

## Pages

```http
GET  /std/cps/inqire/LctreEvlStdPage.do
GET  /std/cps/inqire/LctreEvlResultStdPage.do
POST /std/cps/inqire/LctreEvlViewStdPage.do
```

`LctreEvlViewStdPage.do`는 목록에서 선택한 과목 정보를 form body로 넘겨 여는 HTML 페이지다.

```json
{
  "thisYear": "2026",
  "hakgi": "1",
  "judgeChasu": "middle",
  "judgeChasuName": "",
  "termYn": "Y",
  "openGrade": "<open-grade>",
  "openGwamokNo": "<open-gwamok-no>",
  "openMajorCode": "<open-major-code>",
  "gwamokKname": "<course-name>",
  "bunbanNo": "<bunban-no>",
  "engOpt": "<eng-opt>"
}
```

## List Flow

```http
POST /std/cps/inqire/LctreEvlTermCheck.do
POST /std/cps/inqire/LctreEvlGetHakjuk.do
POST /std/cps/inqire/LctreEvlsugangList.do
```

### `LctreEvlTermCheck.do`

요청 body는 없다.

응답 key:

```json
{
  "thisYear": "2026",
  "hakgi": "1",
  "judgeChasu": "middle",
  "fromDate": "20260601",
  "toDate": "20260630"
}
```

관찰한 화면 로직:

- 빈 응답이면 `수업평가 기간이 아닙니다`.
- 2개 이상 반환되면 평가기간 중복 오류로 처리한다.
- `judgeChasu === "middle"`이면 `중간평가`, `judgeChasu === "last"`이면 `기말평가`.

### `LctreEvlGetHakjuk.do`

요청 body는 없다.

응답 key:

```json
{
  "hakbun": "<student-id>",
  "kname": "<student-name>",
  "gubun": "학부",
  "grade": 2,
  "hakjukStat": "<status>",
  "codeName1": "<department-label>",
  "graduateDate": null
}
```

### `LctreEvlsugangList.do`

요청:

```json
{
  "thisYear": "2026",
  "hakgi": "1",
  "judgeChasu": "middle",
  "judgeChasuName": "",
  "termYn": "Y"
}
```

성공 응답은 배열이다.

```json
[
  {
    "thisYear": "2026",
    "hakgi": "1",
    "openMajorCode": "<open-major-code>",
    "openGrade": "<open-grade>",
    "openGwamokNo": "<open-gwamok-no>",
    "bunbanNo": "<bunban-no>",
    "gwamokKname": "<course-name>",
    "memberName": "<professor-name>",
    "hakjumNum": 3,
    "isuGubun": "<course-type>",
    "engIsuGubun": "<engineering-type>",
    "engOpt": "N",
    "judgeOpt": "N"
  }
]
```

`judgeOpt` 기준:

- `N`: 현재 차수 평가 가능
- `Y`: 평가 완료

## Evaluation View Flow

```http
POST /std/cps/inqire/lctreEvlCheck.do
POST /std/cps/inqire/LctreEvlGetque.do
POST /std/cps/inqire/LctreEvlBunbanCheck.do
POST /std/cps/inqire/LctreEvlBunbanSmallCheck.do
POST /std/cps/inqire/LctreEvlEngQuestion.do
```

모두 `LctreEvlViewStdPage.do`를 열 때 사용한 과목 식별 payload를 그대로 사용한다.

### `lctreEvlCheck.do`

평가 대상자 확인 API.

관찰 응답:

```json
[
  {
    "hakbun": "<student-id>"
  }
]
```

빈 배열이면 평가 대상자가 아닌 것으로 처리한다.

### `LctreEvlGetque.do`

일반 수업평가 문항 API.

응답은 객체다. 관찰 key:

```text
thisYear, hakgi,
q01..q40, q01Eval..q40Eval,
sq01..sq21, sq01Eval..sq20Eval,
fq01,
tq01, tq01Eval, tq02, tq02Eval,
chamgo, chamgoEval, chamgo2, chamgo2Eval, chamgo3
```

화면에서 `q*`는 공통 문항, `sq*`는 선택 문항, `chamgo*`/`fq*`/`tq*`는 서술형 또는 특수 문항으로 사용한다.

### `LctreEvlBunbanCheck.do`

분반별 특수 문항 체크 API.

관찰 케이스에서는 빈 body를 반환했다. 빈 body는 실패로 단정하지 않는다.

### `LctreEvlBunbanSmallCheck.do`

소규모 강좌 체크 API.

관찰 케이스에서는 JSON 문자열을 반환했다.

### `LctreEvlEngQuestion.do`

공학인증 과목 문항 API.

관찰 케이스에서는 빈 body를 반환했다. `engOpt`가 공학인증 대상일 때 `level1..level20`, `studyResult1..studyResult20` 계열 필드가 내려올 가능성이 있다.

## Submit Endpoints

```http
POST /std/cps/inqire/insertEvl.do
POST /std/cps/inqire/insertEng.do
```

주의: 위 API는 실제 평가를 저장하는 side-effect API라 fixture 수집 중 호출하지 않았다.

저장 payload는 과목 식별 payload에 화면의 답변 필드를 추가하는 구조로 보인다.

관찰한 답변 필드 후보:

```text
a01..a40, sa01..sa21, fa01, ta01, ta02,
ea1..ea20, ea21..ea220, ea31..ea320
```

저장 전 화면 검증:

- 모든 필수 선택 문항을 작성해야 한다.
- 일반 서술형 문항은 10자 이상이어야 한다.
- 원격수업 서술형 문항도 10자 이상이어야 한다.
- 공학인증 문항이 있으면 `ea*` 계열 답변을 요구한다.

### 자동 답변 규칙

`klap evaluation submit` 자동화는 다음 값을 사용한다.

- 일반 선택형 문항 `a01..a40`: `5` (`매우 그렇다`/최상위 긍정)
- 분반/특수 선택형 문항 `sa01..sa21`, `fa01`, `ta01`, `ta02`: `5` (`매우 그렇다`/최상위 긍정)
- `chamgo2Opt`: `N` (`기타2. 담당교수가 성차별, 인종차별적 언어사용이나 행동을 하였습니까?` → `아니오`)
- 서술형 문항: `많은 도움 되었습니다. 한학기동안 감사했습니다.`
- 공학인증 추가 문항: 기본 제외. 일반 수업평가는 처리하고, 명시 옵션이 있을 때만 `insertEng.do`까지 호출한다.

실제 제출 API는 `--yes`가 있을 때만 호출하고, 기본 실행은 미리보기로 둔다.

## Result Page

```http
GET /std/cps/inqire/LctreEvlResultStdPage.do
```

관찰 시점에는 결과 확인 기간이 아니어서 JSON API를 호출하지 않았다.
페이지에 `axios.post(...)` 호출은 확인되지 않았고, 결과 확인 기간이 아닐 때 alert를 표시한다.

## Success Fixtures

- `payloads/lecture-evaluation-term.active.json`
- `payloads/lecture-evaluation-term.inactive.txt`
- `payloads/lecture-evaluation-student.success.json`
- `payloads/lecture-evaluation-courses.success.json`
- `payloads/lecture-evaluation-courses.empty.json`
- `payloads/lecture-evaluation-questions.success.json`
- `payloads/lecture-evaluation-target.success.json`
- `payloads/lecture-evaluation-bunban.empty.txt`
- `payloads/lecture-evaluation-small-course.string.json`
- `payloads/lecture-evaluation-engineering.empty.txt`

## Failure Fixtures

- `payloads/lecture-evaluation.session-expired.html`
- `payloads/lecture-evaluation-term.duplicate.json`
- `payloads/lecture-evaluation-courses.template-page.html`
- `payloads/lecture-evaluation-submit.business-error.json`

## Parser Assertions

- `LctreEvlTermCheck.do`가 빈 응답이면 수업평가 기간 외 상태다.
- `LctreEvlsugangList.do`는 배열만 성공으로 본다. HTML이 내려오면 payload 누락 또는 세션/페이지 오류다.
- `LctreEvlGetque.do`는 객체만 성공으로 본다.
- `LctreEvlBunbanCheck.do`, `LctreEvlEngQuestion.do`의 빈 body는 관찰된 정상 케이스다.
- `insertEvl.do`, `insertEng.do`는 명시적 사용자 확인 없이 호출하지 않는다.

## Redaction Rules

- 학번, 이름, 과목명, 교수명, 학정 식별자는 치환한다.
- 평가 문항 본문은 저장할 수 있지만, 특정 학생 답변은 저장하지 않는다.
- 제출 API payload에는 실제 답변을 fixture로 남기지 않는다.
