# KLAP KLAS API 정리

이 문서는 광운대학교 KLAS를 대체하는 macOS 앱 `KLAP` 구현을 위한 비공식 API 메모다.
현재 확인된 내용은 `/Users/wonny/src/Project/Personal/KLAS_video_downloader`의 JS 구현, KLAS 웹 페이지 스크립트 캡처, 실제 로그인 세션 기반 API 호출을 근거로 정리했다.
과제, 공지사항, 시간표는 실제 응답 구조와 앱에서 쓰는 필드를 확인했다. 첨부 파일 계열 API와 제출/업로드 계열 API는 추가 캡처가 필요하다.

## 목표 기능

- KLAS 로그인
- 수업 목록 조회
- 강의 수강
- 강의 다운로드
- 강의 페이지로 이동
- 과제 목록 조회
- 과제 마감 기한 확인
- 과제 페이지로 이동
- 수업별 공지사항 조회
- 공지사항 페이지로 이동
- 시간표 확인
- 과제 자동 미리알림 등록
- 사용자가 설정한 주기마다 KLAS를 조회해 과제와 공지사항 업데이트

## 앱 구현 메모

- 세션 쿠키는 macOS Keychain 또는 암호화 저장소에 저장한다.
- 과제 마감일은 EventKit Reminders에 자동 등록한다.
- 백그라운드 갱신 주기는 사용자 설정값으로 두고, KLAS 서버 부하를 피하기 위해 최소 간격을 둔다.
- KLAS 웹 페이지 이동은 앱 내부 WebView 또는 기본 브라우저 딥링크로 처리한다.
- 서버가 불안정한 상황을 고려해 강의 다운로드는 이어받기, 실패 재시도, 파일 무결성 상태를 저장해야 한다.

## 기본 도메인

- KLAS: `https://klas.kw.ac.kr`
- KWCommons: `https://kwcommons.kw.ac.kr`

## 공통 헤더

### JSON 요청

```http
Content-Type: application/json;charset=utf-8
X-Requested-With: XMLHttpRequest
Cookie: SESSION=<session>; WMONID=<wmonid>;
```

### Form 요청

```http
Content-Type: application/x-www-form-urlencoded; charset=UTF-8
Cookie: SESSION=<session>; WMONID=<wmonid>;
```

## 공통 실패 응답

KLAS는 인증/세션 실패도 HTTP status `200`으로 반환하는 경우가 있다. 따라서 앱은 HTTP status만으로 성공 여부를 판단하면 안 되고, JSON body의 실패 플래그를 먼저 확인해야 한다.

## 성공/실패 케이스 점검 결과

이 문서에 명시된 호출 API는 모두 성공 조건과 실패/예외 처리 기준을 포함한다. `GET *Page.do` 계열은 원문 페이지 열기 성공/실패만 정의하고, 네이티브 데이터 로드는 대응하는 JSON API의 성공/실패 기준을 따른다.

| 영역 | API | 성공/실패 케이스 |
| --- | --- | --- |
| 인증 | `LoginSecurity.do`, `LoginConfirm.do` | 명시됨 |
| 수업 | `YearhakgiAtnlcSbjectList.do`, `LctrumHomeStdInfo.do` | 명시됨 |
| 강의 | `SelectOnlineCntntsStdList.do`, `OnlineCntntsStdPage.do` | 명시됨 |
| 자동 수강 | `LctreCntntsViewSpvPage.do`, `ChkLctreCntntsView.do`, `UpdateProgress.do` | 명시됨 |
| 학습 상태/인증 | `SelectLrnSttusStd.do`, `CertiStdCheck.do`, `SaveLrnStatus.do` | 명시됨 |
| 다운로드 | KWCommons `content.php`, `slide_list_*.xml` | 명시됨 |
| 과제 | `TaskStdPage.do`, `TaskStdList.do`, `TaskStdView.do`, `TaskViewStdPage.do`, `TaskInsertStdPage.do` | 명시됨 |
| 팀프로젝트/퀴즈 | `PrjctStdList.do`, `AnytmQuizStdList.do` | 명시됨 |
| 공지 | `BoardListStdPage.do`, `BoardStdList.do`, `BoardStdView.do`, `BoardViewStdPage.do`, `LctrumSchdulInfo.do` | 명시됨 |
| 시간표 | `TimetableStdPage.do`, `AtnlcYearList.do`, `TimetableStdList.do` | 명시됨 |
| 세션 연장 | `UpdateSession.do` | 명시됨 |
| 강의계획서 보조 | `CmmnGamokList.do`, `CmmnHakgwaList.do`, `CmmnMagerCodeList.do`, `CultureOptOneInfo.do`, `LectrePlanStdCrtNum.do`, `LectrePlanDaList.do` | 명시됨 |
| Everytime | `find/lecture/list/keyword`, `find/lecture` | 명시됨 |

### 공통 성공 판정

JSON API는 아래 조건을 모두 만족할 때만 성공으로 본다.

- HTTP status가 `2xx`다.
- 응답 body가 해당 API의 기대 타입이다.
- `loginRequired !== true`다.
- `errorCount`가 없거나 `0`이다.
- 목록 API는 기대 배열 또는 기대 목록 key를 포함한다.
- 상세 API는 본문 렌더링에 필요한 핵심 객체를 포함한다.

HTML 또는 문자열 API는 아래 조건을 만족할 때 성공으로 본다.

- HTTP status가 `2xx`다.
- 응답 body가 로그인 페이지나 공통 오류 페이지가 아니다.
- HTML API는 앱이 필요한 form/script 또는 원문 페이지 렌더링에 필요한 HTML을 포함한다.
- 문자열 API는 빈 문자열이 아니고, 앱이 표시할 수 있는 형식이다.

## 응답 구조 표기 규칙

- 실제 응답 전체를 문서에 그대로 저장하지 않는다.
- 사용자 식별자, 과목명, 과제/공지 본문, 파일명, 세션 값은 placeholder로 표기한다.
- 배열 응답은 첫 번째 항목의 확인된 key와 앱이 사용하는 필드를 중심으로 적는다.
- KLAS가 HTML 또는 문자열을 반환하는 API는 JSON처럼 파싱하지 않고 응답 타입과 앱 사용 지점만 적는다.
- `*Page.do` 계열 GET은 대부분 KLAS 전체 HTML 페이지를 반환한다. 앱의 네이티브 데이터 소스는 `*List.do`, `*View.do`, `*Info.do` 계열 JSON API를 우선 사용한다.

### HTTP/네트워크/파싱 실패

모든 API에 공통 적용한다.

- HTTP status가 `4xx`, `5xx`면 실패로 처리한다.
- JSON API에서 HTML이 내려오면 세션 만료 또는 KLAS 오류 페이지로 보고 실패 처리한다.
- HTML API에서 로그인 페이지가 내려오면 세션 만료로 처리한다.
- 네트워크 timeout, DNS 실패, TLS 실패, 연결 중단은 서버 응답 실패와 구분해 재시도 가능한 네트워크 오류로 표시한다.
- 기대 타입이 배열인데 객체가 내려오거나, 객체인데 배열/문자열이 내려오면 API 구조 변경 가능성으로 보고 진단 로그를 남긴다.

### 로그인 필요 또는 세션 만료

세션 쿠키가 없거나 만료된 상태로 인증이 필요한 JSON API를 호출하면 HTTP status는 `200`이고, body는 아래 구조로 내려온다.

```json
{
  "redirect": true,
  "redirectUrl": "/usr/cmn/login/LoginForm.do?redirectUrl=/std/cmn/frame/YearhakgiAtnlcSbjectList.do",
  "fieldErrors": [],
  "responseText": "",
  "response": null,
  "loginRequired": true,
  "errorCount": 0
}
```

앱 처리:

- `loginRequired === true`면 현재 세션을 실패로 보고 재로그인 또는 로그인 화면 이동을 수행한다.
- `redirectUrl`은 사용자를 웹 로그인 페이지로 보낼 때만 사용한다.
- 이 응답은 HTTP 오류가 아니므로 `URLSession`/axios의 status 검사만으로는 잡히지 않는다.

### 입력/업무 오류

업무 오류는 `errorCount > 0`과 `fieldErrors`로 내려올 수 있다.

```json
{
  "redirectUrl": "",
  "fieldErrors": [
    {
      "field": "",
      "message": "<오류 메시지>"
    }
  ],
  "responseText": "",
  "response": {},
  "redirect": false,
  "errorCount": 1,
  "loginRequired": false
}
```

앱 처리:

- `errorCount > 0`이면 실패로 처리한다.
- 사용자에게 보여줄 메시지는 `fieldErrors[0].message`를 우선 사용한다.
- `fieldErrors`가 비어 있으면 일반 오류 메시지를 사용한다.

## 인증

### 로그인 보안 정보 조회

```http
POST /usr/cmn/login/LoginSecurity.do
```

응답에서 사용하는 값:

- `Set-Cookie`: `SESSION`, `WMONID`
- JSON body: `publicKey`

확인된 응답 구조:

```json
{
  "publicKey": "<rsa-public-key-body>"
}
```

성공 조건:

- HTTP status `2xx`
- `Set-Cookie`에 `SESSION` 또는 `WMONID` 후보가 포함됨
- JSON body에 비어 있지 않은 `publicKey`가 포함됨

실패/예외:

- `publicKey`가 없거나 빈 문자열이면 로그인 토큰을 만들 수 없으므로 로그인 시작 실패로 처리한다.
- `Set-Cookie`가 없으면 `LoginConfirm.do`에서 같은 세션을 이어갈 수 없으므로 재시도 후 실패 처리한다.
- JSON 대신 HTML이 내려오면 KLAS 로그인 페이지/오류 페이지로 보고 재시도 가능한 서버 오류로 처리한다.
- 공개키 PEM 복원 또는 RSA 암호화가 실패하면 서버 호출 전 클라이언트 암호화 오류로 처리한다.

`publicKey`는 PEM 헤더와 푸터 없이 내려온다. 앱에서는 다음 형식으로 복원한다.

```text
-----BEGIN PUBLIC KEY-----
<publicKey>
-----END PUBLIC KEY-----
```

### 로그인 확인

```http
POST /usr/cmn/login/LoginConfirm.do
```

요청 헤더: JSON 요청

요청 body:

```json
{
  "loginToken": "<rsa_pkcs1_base64>",
  "redirectUrl": "",
  "redirectTabUrl": ""
}
```

`loginToken` 생성 payload:

```json
{
  "loginId": "<student_id>",
  "loginPwd": "<password>",
  "storeIdYn": "N"
}
```

생성 방식:

- `LoginSecurity.do`에서 받은 공개키 사용
- RSA PKCS#1 v1.5 padding으로 암호화
- 암호화 결과를 Base64 인코딩

응답 사용 방식:

- 로그인 성공 시 `errorCount`는 `0`이다.
- 로그인 실패도 HTTP status `200`으로 내려올 수 있으므로 반드시 body를 확인한다.
- `errorCount > 0`이면 `fieldErrors[0].message`를 로그인 실패 메시지로 사용한다.
- 성공 후에는 `LoginSecurity.do`와 `LoginConfirm.do`에서 받은 `SESSION`, `WMONID` 쿠키를 이후 요청에 재사용한다.

성공 조건:

- HTTP status `2xx`
- `loginRequired === false`
- `errorCount === 0`
- `response.userId`가 존재함

로그인 성공 응답 구조:

```json
{
  "redirectUrl": "",
  "fieldErrors": [],
  "responseText": "",
  "response": {
    "frstPwdAt": null,
    "pushToken": null,
    "userId": "<student-or-user-id>"
  },
  "errorCount": 0,
  "redirect": false,
  "loginRequired": false
}
```

로그인 실패 응답:

```json
{
  "redirectUrl": "",
  "fieldErrors": [
    {
      "field": "",
      "message": "개인번호 또는 비밀번호가 일치하지 않습니다."
    }
  ],
  "responseText": "",
  "response": {},
  "redirect": false,
  "errorCount": 1,
  "loginRequired": false
}
```

실패/예외:

- `errorCount > 0`이면 인증 실패 또는 업무 오류로 처리한다.
- `fieldErrors[0].message`가 있으면 그대로 로그인 실패 메시지에 사용한다.
- `response.userId`가 없으면 성공 응답처럼 보여도 로그인 세션 검증 실패로 처리한다.
- 로그인 성공 후 다음 인증 API에서 `loginRequired === true`가 내려오면 쿠키 저장 실패 또는 세션 불일치로 보고 쿠키를 폐기한다.

## 수업 목록

### 학기별 수강 과목 목록 조회

```http
POST /std/cmn/frame/YearhakgiAtnlcSbjectList.do
```

요청 헤더: JSON 요청

요청 body:

```json
{}
```

응답 사용 방식:

- JS 구현은 `res.data.at(0)`을 최신 학기로 사용한다.
- 최신 학기 객체의 `value`를 `yearHakgi`로 사용한다.
- 최신 학기 객체의 `subjList`를 수업 목록으로 사용한다.
- 각 `subjList` 항목의 `name`은 UI 표시용 수업명으로 사용한다.
- 각 `subjList` 항목의 `value`는 과목 컨텍스트 변경, 과제/공지/시간표 조회의 `selectSubj` 값으로 사용한다.

확인된 응답 구조:

```json
[
  {
    "label": "2026년도 1학기",
    "value": "2026,1",
    "subjList": [
      {
        "name": "수업명",
        "value": "과목 선택값"
      }
    ]
  }
]
```

성공 조건:

- 루트가 배열이다.
- 하나 이상의 학기 객체가 있고, 최신 학기 객체에 `value`와 `subjList`가 있다.
- `subjList`가 배열이며 각 과목에 `name`, `value`가 있다.

실패/예외:

- 공통 실패 응답 또는 로그인 페이지 HTML이 내려오면 세션 만료로 처리한다.
- 루트 배열이 비어 있으면 수강 과목 없음 상태로 표시한다.
- `subjList`가 없거나 배열이 아니면 수업 목록 구조 변경으로 보고 동기화를 실패 처리한다.
- 특정 과목의 `name` 또는 `value`가 없으면 그 과목만 제외하고 진단 로그에 누락 필드를 남긴다.

### 과목 컨텍스트 변경

```http
POST /std/lis/evltn/LctrumHomeStdInfo.do
```

요청 헤더: JSON 요청

요청 body:

```json
{
  "selectYearhakgi": "<yearHakgi>",
  "selectSubj": "<subject.value>",
  "selectChangeYn": "Y"
}
```

이 요청 이후 온라인 강의 목록 조회가 선택한 과목 기준으로 동작한다.

확인된 응답 구조:

```json
{
  "noticeList": [
    {
      "boardNo": 1161280,
      "masterNo": 1000000,
      "grcode": "<grcode>",
      "year": "2026",
      "hakgi": "1",
      "subj": "<subject-id>",
      "bunban": "01",
      "title": "공지 제목",
      "content": "<html-or-text>",
      "topAt": "N",
      "othbcAt": "Y",
      "atchFileId": null,
      "readCnt": 0,
      "userNm": "작성자",
      "registDt": "2026-05-29T02:20:00.000+09:00",
      "dateDiff": 0,
      "fileCnt": 0,
      "cmCnt": 0
    }
  ],
  "cntntList": [],
  "rtprgsList": [],
  "onRtprgsList": [],
  "atendList": [],
  "atendSubList": [],
  "taskTop": [],
  "prjctTop": [],
  "examTop": [],
  "anQuizTop": [],
  "dscsnTop": [],
  "taskCnt": 0,
  "taskPrsntCnt": 0,
  "taskNewCnt": 0,
  "prjctCnt": 0,
  "prjctPrsntCnt": 0,
  "prjctNewCnt": 0,
  "examCnt": 0,
  "examPrsntCnt": 0,
  "examNewCnt": 0,
  "quizCnt": 0,
  "quizPrsntCnt": 0,
  "quizNewCnt": 0,
  "surveyCnt": 0,
  "surveyPrsntCnt": 0,
  "surveyNewCnt": 0,
  "dscsnCnt": 0,
  "dscsnJoinCnt": 0,
  "dscsnNewCnt": 0,
  "pdsCnt": 0,
  "pdsNewCnt": 0,
  "cntntCmpltCnt": 0,
  "isonoff": "ON"
}
```

응답 사용 방식:

- JS 구현은 응답 body를 사용하지 않는다.
- HTTP 오류가 없으면 서버 세션의 현재 과목 컨텍스트가 변경된 것으로 본다.
- 이후 `SelectOnlineCntntsStdList.do` 같은 과목 의존 API를 호출한다.

성공 조건:

- HTTP status `2xx`
- 공통 실패 응답이 아님
- 응답이 객체이며 `loginRequired !== true`, `errorCount`가 없거나 `0`

실패/예외:

- `loginRequired === true`면 세션 만료로 처리한다.
- `errorCount > 0`이면 과목 변경 실패로 처리하고 `fieldErrors[0].message`를 우선 표시한다.
- HTTP 오류 또는 JSON 파싱 실패가 발생하면 이후 과목 의존 API를 호출하지 않는다.
- 과목 변경 직후 조회한 목록이 요청한 과목과 명백히 다르면 컨텍스트 불일치로 보고 해당 과목 동기화를 실패 처리한다.

## 온라인 강의

### 온라인 강의 목록 조회

```http
POST /std/lis/evltn/SelectOnlineCntntsStdList.do
```

요청 헤더: JSON 요청

요청 body:

```json
{}
```

응답 사용 방식:

- 강의 목록 배열을 반환한다.
- `evltnSe === "proj"`인 항목은 과제 항목으로 보고 강의 목록에서 제외한다.
- 자동 수강 대상은 `prog`가 있고, `prog !== 100`이며, 현재 시간이 수강 가능 기간 안인 항목만 사용한다.
- `mvpLink` 또는 `starting`은 선택 강의 재생 URL로 우선 사용한다.
- 두 필드에서 `em/<contentId>` 형태만 확인되는 경우 `https://kwcommons.kw.ac.kr/em/<contentId>`로 보정해 재생 링크를 만든다.
- 다운로드에는 같은 필드에서 KWCommons 콘텐츠 ID를 추출해 다운로드 URL 조회에 사용한다.
- 뷰어/진도 API 호출에는 같은 강의 객체의 식별 필드들을 그대로 전달한다.
- `sdateY/sdateH/sdateM`, `edateY/edateH/edateM`은 앱 내부에서 `Date`로 조합해 수강 가능 여부와 마감 표시를 계산한다.

확인된 응답 구조:

```json
[
  {
    "userId": "<user-id>",
    "grcode": "<grcode>",
    "subj": "<subject-id>",
    "year": "2026",
    "hakgi": "1",
    "bunban": "01",
    "module": "01",
    "lesson": "001",
    "sdesc": "강의 설명",
    "evltnSe": "on",
    "sbjt": "강의 제목",
    "rcognTime": "50",
    "weeklyseq": "1",
    "weekNo": "1",
    "moduletitle": "1주차",
    "sdate": "202603060000",
    "edate": "202603122359",
    "today": "202605290332",
    "startDate": "2026-03-06 00:00",
    "endDate": "2026-03-12 23:59",
    "sdateY": "20260306",
    "sdateH": "00",
    "sdateM": "00",
    "edateY": "20260312",
    "edateH": "23",
    "edateM": "59",
    "oid": "C000000000",
    "connYn": "N",
    "ispreview": "N",
    "isonoff": "ON",
    "ptype": "V",
    "ptime": "50",
    "totalTime": "0",
    "learnTime": "0",
    "achivTime": "0",
    "prog": 0,
    "lrnPd": "2026-03-06 00:00 ~ 2026-03-12 23:59",
    "starting": "https://kwcommons.kw.ac.kr/em/<content-id>",
    "mvpLink": "https://kwcommons.kw.ac.kr/em/<content-id>&contents=...",
    "width": "9999",
    "height": 1080,
    "types": "V",
    "bunbanroom": null,
    "pcond1": null,
    "pcond2": null,
    "sdate2": null,
    "edate2": null,
    "totRcognTime": "50",
    "totAchivTime": "0"
  }
]
```

JS 구현에서 사용하는 강의 필드:

| 필드 | 용도 |
| --- | --- |
| `moduletitle` | 주차 또는 모듈 제목 |
| `sbjt` | 강의 제목 |
| `mvpLink` | KWCommons 재생 URL, 콘텐츠 ID 추출 |
| `starting` | KWCommons 재생 URL 대체값, 콘텐츠 ID 대체 추출 |
| `evltnSe` | 강의/과제 구분 |
| `prog` | 수강 진도 |
| `sdateY`, `sdateH`, `sdateM` | 수강 시작 일시 |
| `edateY`, `edateH`, `edateM` | 수강 종료 일시 |
| `grcode`, `subj`, `year`, `hakgi`, `bunban`, `module`, `oid` | 뷰어/진도 API 파라미터 |
| `ptime`, `totalTime`, `weekNo`, `weeklyseq`, `lesson`, `ispreview` | 뷰어/진도 API 파라미터 |

성공 조건:

- 루트가 배열이다.
- 강의 항목은 `evltnSe`, `sbjt`, `prog`, 기간 필드 중 앱 표시/수강 판단에 필요한 값을 포함한다.
- 배열이 비어 있으면 수강 가능한 온라인 강의 없음 상태로 처리한다.

실패/예외:

- 공통 실패 응답 객체가 내려오면 세션 만료 또는 업무 오류로 처리한다.
- 루트가 배열이 아니면 API 구조 변경으로 보고 강의 목록 로드를 실패 처리한다.
- 과목 컨텍스트 변경 없이 호출해 다른 과목 데이터가 내려오면 현재 선택 과목과 일치하지 않는 항목은 제외한다.
- `evltnSe === "proj"` 항목은 과제이므로 강의 목록에서 제외한다. 이 제외는 실패가 아니다.
- 기간 필드가 없거나 파싱할 수 없으면 해당 강의의 수강 가능 여부를 `확인 필요`로 표시하고 자동 수강 대상에서는 제외한다.
- `mvpLink`, `starting` 모두 없고 뷰어 호출에 필요한 식별 필드도 부족하면 재생/다운로드 불가 강의로 표시한다.

### 강의 페이지 열기

```http
GET /std/lis/evltn/OnlineCntntsStdPage.do
```

용도:

- KLAS 웹에서 "온라인 강의컨텐츠 보기" 메뉴가 여는 페이지다.
- 페이지 내부에는 `viewForm`이 있으며, 강의 보기 버튼은 아래 필드를 채운 뒤 뷰어 페이지로 이동한다.

`viewForm` 필드:

```json
{
  "grcode": "",
  "subj": "",
  "year": "",
  "hakgi": "",
  "bunban": "",
  "module": "",
  "lesson": "",
  "oid": "",
  "ptime": "",
  "weeklyseq": "",
  "weeklysubseq": "",
  "totalTime": "",
  "prog": "",
  "profYN": "N",
  "previewYN": "N",
  "late": "N"
}
```

비고:

- 앱에서 특정 강의를 열 때는 먼저 과목 컨텍스트를 맞춘 뒤 `OnlineCntntsStdPage.do` 또는 뷰어 API를 사용한다.
- 실제 온라인 강의 목록 데이터는 앞의 `SelectOnlineCntntsStdList.do`에서 가져오는 편이 앱 구현에 적합하다.
- 사용자가 앱에서 특정 강의를 재생할 때는 이 목록 페이지보다 `mvpLink`, `starting`의 KWCommons URL을 우선 연다.
- 재생 URL을 만들 수 없는 강의에 한해 `LctrumHomeStdInfo.do`로 과목 컨텍스트를 맞춘 뒤 이 목록 페이지로 폴백한다.

응답 사용 방식:

- 응답은 HTML 페이지다.
- 앱에서 WebView로 강의 페이지를 띄울 때는 이 HTML을 그대로 표시한다.
- 네이티브 UI에서는 HTML 전체를 파싱하기보다 `SelectOnlineCntntsStdList.do` 응답을 데이터 소스로 사용한다.

성공 조건:

- HTTP status `2xx`
- 로그인 페이지가 아닌 HTML 페이지가 내려옴
- WebView 폴백 표시 또는 원문 페이지 이동에 사용할 수 있음

실패/예외:

- 로그인 페이지 HTML이 내려오면 세션 만료로 처리한다.
- HTML 로드는 성공했지만 강의 목록 데이터는 포함하지 않을 수 있으므로 네이티브 목록 갱신 성공으로 간주하지 않는다.
- 페이지 로드 실패 시 `SelectOnlineCntntsStdList.do` 캐시 데이터는 유지하고 원문 페이지 열기만 실패로 표시한다.

## 강의 자동 수강

### 뷰어 페이지 호출 및 `lecKey` 획득

```http
POST /spv/lis/lctre/viewer/LctreCntntsViewSpvPage.do
```

요청 헤더: Form 요청

요청 body:

```json
{
  "grcode": "<lesson.grcode>",
  "subj": "<lesson.subj>",
  "year": "<lesson.year>",
  "hakgi": "<lesson.hakgi>",
  "bunban": "<lesson.bunban>",
  "module": "<lesson.module>",
  "oid": "<lesson.oid>",
  "ptime": "<lesson.ptime>",
  "weeklyseq": "<lesson.weekNo>",
  "weeklysubseq": "<lesson.weeklyseq>",
  "totalTime": "<lesson.totalTime>",
  "prog": "<lesson.prog>",
  "lesson": "<lesson.lesson>",
  "profYN": "Y",
  "previewYN": "<lesson.ispreview>",
  "late": "N"
}
```

응답 HTML 또는 스크립트에서 다음 정규식으로 `lecKey`를 추출한다.

```regex
"lecKey"\s*:\s*'([^']+)'
```

확인된 응답 구조:

```html
<!DOCTYPE html>
<html>
  <head>...</head>
  <body>
    ...
    <script>
      /* 플레이어 초기화 스크립트 내부 */
      "lecKey": "<lecture-key>"
    </script>
  </body>
</html>
```

응답 사용 방식:

- 응답은 HTML 또는 JavaScript가 포함된 페이지다.
- 자동 수강 구현은 `lecKey`만 추출해 `ChkLctreCntntsView.do`, `UpdateProgress.do` body에 추가한다.
- `lecKey`를 찾지 못하면 해당 강의의 자동 수강을 중단하고 사용자에게 재시도 또는 WebView 수강을 안내한다.

성공 조건:

- HTTP status `2xx`
- 로그인 페이지가 아닌 HTML/스크립트 응답
- 응답 본문에서 `lecKey`를 추출할 수 있음

실패/예외:

- 요청 body에 강의 식별 필드가 부족하면 호출하지 않고 강의 데이터 오류로 처리한다.
- 로그인 페이지 또는 공통 오류 HTML이 내려오면 세션 만료 또는 서버 오류로 처리한다.
- `lecKey`가 없으면 이후 수강 상태 확인/진도 업데이트를 호출하지 않는다.
- `weekNo`, `weeklyseq`, `lesson` 매핑이 잘못되면 HTTP 200이어도 `lecKey`가 누락될 수 있으므로 해당 강의 자동 수강만 실패 처리한다.

### 수강 상태 확인

```http
POST /spv/lis/lctre/viewer/ChkLctreCntntsView.do
```

요청 헤더: Form 요청

요청 body:

```json
{
  "grcode": "<lesson.grcode>",
  "subj": "<lesson.subj>",
  "year": "<lesson.year>",
  "hakgi": "<lesson.hakgi>",
  "bunban": "<lesson.bunban>",
  "module": "<lesson.module>",
  "oid": "<lesson.oid>",
  "ptime": "<lesson.ptime>",
  "weeklyseq": "<lesson.weekNo>",
  "weeklysubseq": "<lesson.weeklyseq>",
  "lesson": "<lesson.lesson>",
  "lecKey": "<lecKey>"
}
```

현재 JS 구현은 진도 업데이트 직전에 이 요청을 호출한다.

응답 구조:

```json
{}
```

응답 사용 방식:

- JS 구현은 응답 body를 사용하지 않는다.
- HTTP 오류가 없으면 수강 상태 확인이 통과한 것으로 보고 곧바로 `UpdateProgress.do`를 호출한다.
- 앱은 이 응답 body를 파싱하지 않고, HTTP status와 공통 실패 응답만 확인한다.

성공 조건:

- HTTP status `2xx`
- 공통 실패 응답이 아님
- 빈 객체 `{}` 또는 사용하지 않는 객체 응답

실패/예외:

- `lecKey`가 없으면 호출하지 않는다.
- HTTP 오류, 공통 실패 응답, 로그인 페이지 HTML이 내려오면 진도 업데이트를 호출하지 않는다.
- 빈 객체가 아닌 오류 객체가 내려오면 `errorCount`, `loginRequired`, `fieldErrors`를 확인해 실패로 처리한다.

### 수강 진도 업데이트

```http
POST /spv/lis/lctre/viewer/UpdateProgress.do
```

요청 헤더: Form 요청

요청 body: `ChkLctreCntntsView.do`와 동일

확인된 응답 구조:

```json
{
  "data": {
    "totalTime": "10",
    "ptime": "50",
    "prog": 20
  }
}
```

일부 응답은 아래처럼 루트에 진행 필드가 올 수 있으므로 앱은 양쪽을 모두 허용한다.

```json
{
  "totalTime": "10",
  "ptime": "50",
  "prog": 20
}
```

응답 사용 방식:

- HTTP status `200`을 기대한다.
- `res.data.totalTime`, `res.data.ptime`으로 진행 상태를 표시한다.
- `res.data.prog === 100`이면 수강 완료로 본다.
- 기존 JS 구현은 60초마다 반복 호출한다.
- `totalTime`은 현재 누적 수강 시간, `ptime`은 인정에 필요한 재생 시간으로 표시한다.
- `prog`가 `100`보다 작으면 다음 주기에도 같은 body로 업데이트를 반복한다.

성공 조건:

- HTTP status `2xx`
- 공통 실패 응답이 아님
- `data.prog` 또는 루트 `prog`가 숫자 또는 숫자 문자열로 존재함
- `data.totalTime`/루트 `totalTime`과 `data.ptime`/루트 `ptime` 중 진행 표시가 가능한 값이 존재함

실패/예외:

- `prog`를 찾을 수 없으면 진도 반영 여부를 알 수 없으므로 해당 업데이트를 실패로 처리한다.
- `prog`가 이전 값보다 감소하면 서버 지연 또는 다른 세션 갱신 가능성이 있으므로 즉시 완료 처리하지 않고 다음 목록 재조회로 검증한다.
- 반복 호출 중 동일한 `prog`가 장시간 유지되면 수강 인정 조건 미충족 또는 서버 반영 지연으로 보고 사용자에게 확인 필요 상태를 표시한다.
- `loginRequired === true`, `errorCount > 0`, HTTP 오류, HTML 응답은 자동 수강 실패로 처리하고 반복을 중단한다.

## 강의 다운로드

### KWCommons 콘텐츠 ID 추출

강의 객체에서 다음 순서로 콘텐츠 ID를 추출한다.

```js
if (lesson.mvpLink) {
  return lesson.mvpLink.split("em/")[1].split("&contents")[0];
}

if (lesson.starting) {
  return lesson.starting.split("em/")[1];
}
```

### 실제 동영상 URL 조회

```http
GET https://kwcommons.kw.ac.kr/viewer/ssplayer/uniplayer_support/content.php?content_id=<lessonId>
```

확인된 응답 구조:

```xml
<content>
  <desktop>
    <media_uri>https://.../video.mp4</media_uri>
  </desktop>
  <mobile>...</mobile>
</content>
```

대체 응답 구조:

```xml
<content>
  <media_uri target="all">https://.../[MEDIA_FILE]</media_uri>
  <main_media media_id="...">video.mp4</main_media>
</content>
```

응답 파싱 방식:

1. `<desktop>` 내부의 `<media_uri>` 값을 사용한다.
2. 실패하면 `target="all">` 뒤의 prefix와 `<main_media media_id=...>` 내부 값을 조합한다.

앱 구현 시 주의할 점:

- 문자열 split 기반 파싱은 취약하므로 XML/HTML 파서를 사용하는 편이 좋다.
- 다운로드 상태는 과목 ID, 강의 ID, 저장 경로, byte progress, 완료 여부로 저장한다.
- 파일명에는 macOS 파일명 금지 문자를 제거한다.

응답 사용 방식:

- KWCommons 응답 자체는 플레이어 설정 문서로 보고, 앱은 최종 미디어 URL만 추출한다.
- 추출한 URL은 별도 인증 헤더 없이 `GET` stream으로 내려받는 방식이 현재 구현에서 확인됐다.
- 다운로드 실패 시에는 마지막 완료 강의 index를 상태 파일에 저장하고 다음 실행에서 이어서 처리한다.

성공 조건:

- HTTP status `2xx`
- XML/HTML 본문에서 최종 미디어 URL을 추출할 수 있음
- 추출한 URL이 `http://` 또는 `https://`로 시작함

실패/예외:

- 강의 객체에서 KWCommons 콘텐츠 ID를 추출할 수 없으면 다운로드 불가로 표시한다.
- KWCommons 응답에 `<media_uri>` 또는 조합 가능한 `<main_media>`가 없으면 미디어 URL 조회 실패로 처리한다.
- 추출한 URL이 상대 경로이거나 스킴이 없으면 KWCommons base URL 기준 보정 후에도 유효하지 않을 때 실패 처리한다.
- 실제 파일 `GET`이 `403`, `404`, `5xx`, timeout이면 다운로드 실패로 기록하고 재시도 정책을 적용한다.
- partial download가 중단되면 byte offset과 임시 파일 상태를 저장하되, 파일 크기 또는 checksum을 확인할 수 없으면 완료로 표시하지 않는다.

## 과제

### 과제 페이지 열기

```http
GET /std/lis/evltn/TaskStdPage.do
```

용도:

- KLAS 웹에서 "과제제출" 메뉴가 여는 페이지다.
- 과목 선택 컴포넌트가 선택한 학기/과목을 기준으로 과제 목록을 조회한다.

성공 조건:

- HTTP status `2xx`
- 로그인 페이지가 아닌 HTML 페이지가 내려옴

실패/예외:

- 로그인 페이지 HTML이 내려오면 세션 만료로 처리한다.
- 네이티브 과제 목록 데이터는 이 페이지가 아니라 `TaskStdList.do`로 판단한다.

### 과제 목록 조회

```http
POST /std/lis/evltn/TaskStdList.do
```

요청 헤더: JSON 요청

요청 body:

```json
{
  "selectYearhakgi": "<yearHakgi>",
  "selectSubj": "<subject.value>",
  "currentPage": 0
}
```

확인된 동작:

- 페이지 스크립트는 `currentPage`를 전송 전에 `1` 감소시킨다.
- 실제 확인된 응답은 루트 배열이다.
- 앱에서는 `ordseq`, `weeklyseq`, `weeklysubseq`, `title`, `expiredate`, `submityn`을 사용한다.

확인된 응답 구조:

```json
[
  {
    "taskNo": 1,
    "ordseq": "7",
    "weeklyseq": "15",
    "weeklysubseq": "1",
    "title": "과제명",
    "startdate": "2026-05-01 00:00:00",
    "expiredate": "2026-06-17 23:59:59",
    "restartdate": null,
    "reexpiredate": null,
    "score": null,
    "isopen": "Y",
    "indate": "Y",
    "adddate": "N",
    "submityn": "N",
    "weight": null
  }
]
```

응답 사용 방식:

- `ordseq`는 과제 상세/제출/우수보고서/전체보고서 페이지 이동에 쓰는 핵심 과제 식별자다.
- `weeklyseq`, `weeklysubseq`는 제출 페이지(`TaskInsertStdPage.do`) 이동에 함께 전달한다.
- `title`은 과제명이다.
- `expiredate`는 마감 시각이며 `yyyy-MM-dd HH:mm:ss` 형식이다.
- `submityn`은 제출 여부이며 `Y`면 제출 완료, `N`이면 미제출이다.
- `startdate`는 과제 시작 시각이다.
- `restartdate`, `reexpiredate`는 재제출 기간 관련 필드로 보이며 실제 응답에서는 `null`인 케이스를 확인했다.
- `indate`, `adddate`, `isopen`은 KLAS UI 상태 표시에 쓰이는 플래그로 보이며 앱의 핵심 목록에는 아직 사용하지 않는다.
- 앱은 과제별 안정 ID를 `subject.value + ordseq` 조합으로 우선 구성할 수 있다.
- 현재 앱 구현은 루트 배열과 위 실제 필드명만 기준으로 과제를 매핑한다.

성공 조건:

- 루트가 배열이다.
- 배열이 비어 있으면 해당 과목 과제 없음 상태로 처리한다.
- 과제 항목은 최소 `ordseq`, `title`을 포함한다.
- `expiredate`, `submityn`은 정상 상태 계산에 필요하지만 누락되어도 항목 자체는 보존한다.

실패/예외:

- 공통 실패 응답 객체 또는 로그인 페이지 HTML이 내려오면 세션 만료/업무 오류로 처리한다.
- 루트가 배열이 아니면 API 구조 변경으로 보고 과제 목록 로드를 실패 처리한다.
- `ordseq`가 없으면 상세/제출 페이지로 이동할 수 없으므로 해당 과제 항목만 제외한다.
- `expiredate`가 없거나 파싱할 수 없으면 목록에는 표시하되 마감 정렬/미리알림 등록 대상에서는 제외하고 `마감 확인 필요`로 표시한다.
- `submityn`이 없으면 제출 상태를 `확인 필요`로 표시하고 홈 화면의 제출 완료 필터에 사용하지 않는다.

### 과제 상세 조회

```http
POST /std/lis/evltn/TaskStdView.do
```

요청 헤더: JSON 요청

요청 body:

```json
{
  "selectYearhakgi": "<yearHakgi>",
  "selectSubj": "<subject.value>",
  "ordseq": "<assignment.ordseq>"
}
```

확인된 응답 구조:

- 루트 객체는 `rpt`, `smt`를 포함한다.
- `rpt`는 출제 과제 정보다.
- `smt`는 내 제출 정보이며 제출 전/채점 전에는 일부 값이 `null`일 수 있다.
- 기존 `GET /std/lis/evltn/TaskViewStdPage.do`는 Angular 템플릿 HTML을 반환하므로 앱 상세 본문 렌더링에 직접 사용하지 않는다.

확인된 응답 예시:

```json
{
  "rpt": {
    "ordseq": "7",
    "weeklyseq": "15",
    "weeklysubseq": "1",
    "title": "과제명",
    "contents": "<p>과제 본문 HTML</p>",
    "startdate": "2026-05-01 00:00:00",
    "expiredate": "2026-06-17 23:59:59",
    "restartdate": null,
    "reexpiredate": null,
    "reptype": "P",
    "submityn": "N",
    "submitfiletype": "상관없음",
    "filelimit": "30",
    "isopen": "Y",
    "isopenscore": "N",
    "attachopen": "Y",
    "basicscore": null,
    "atchFileId": null,
    "realfile": null,
    "newfile": null,
    "indate": "Y",
    "adddate": "N"
  },
  "smt": {
    "seq": null,
    "title": null,
    "contents": null,
    "registDt": null,
    "finalscore": null,
    "submitScore": null,
    "tutorcontents": null,
    "tutordate": null,
    "isbestreport": "N",
    "atchFileId": null,
    "realfile": null,
    "newfile": null,
    "tutorFileId": null,
    "tutorrealfile": null,
    "tutornewfile": null,
    "name": null,
    "userNm": null,
    "hakgwanm": null,
    "projid": null,
    "indate": null
  }
}
```

응답 사용 방식:

- `rpt.title`은 상세 화면 제목이다.
- `rpt.contents`는 과제 본문/주의사항이며 HTML 문자열이므로 앱에서 텍스트로 정리해 표시한다.
- `rpt.startdate`, `rpt.expiredate`는 제출기한 표시용이다.
- `rpt.reptype`는 제출 방식이며 확인된 값 기준 `1` 또는 `P`는 개인별, `2`는 팀별로 표시한다.
- `rpt.submityn`은 제출 상태이며 `Y`면 제출, 그 외는 미제출로 취급한다.
- `rpt.submitfiletype`, `rpt.filelimit`은 제출 양식과 파일 용량 제한 표시용이다.
- `rpt.realfile`, `rpt.newfile`은 과제 첨부 파일명 후보로 표시한다.
- `smt.title`, `smt.contents`는 사용자가 제출한 내용 표시용이다.
- `smt.finalscore`, `smt.tutorcontents`는 채점 결과/의견 표시용이다.
- `smt.realfile`, `smt.newfile`, `smt.tutorrealfile`, `smt.tutornewfile`은 제출 파일 또는 첨삭 파일명 후보로 표시한다.

성공 조건:

- HTTP status `2xx`
- 루트 객체에 `rpt`가 있고 `rpt.title` 또는 `rpt.contents` 중 하나 이상이 존재함
- `smt`는 없거나 일부 필드가 `null`이어도 성공으로 처리 가능

실패/예외:

- 세션이 만료되면 로그인 페이지 HTML 또는 권한 오류 응답이 내려올 수 있으므로 JSON 파싱 실패를 세션 오류로 처리해야 한다.
- `rpt`가 없으면 과제 상세로 볼 수 없는 응답이므로 앱은 상세 로드 실패로 표시한다.
- `ordseq` 없이 호출하면 상세를 특정할 수 없으므로 클라이언트 요청 구성 오류로 처리한다.
- `contents`가 비어 있으면 본문 없음으로 표시하되 상세 로드 자체는 성공으로 둔다.
- 확인된 후보 중 `TaskViewStd.do`, `TaskStdInfo.do`, `TaskViewStdInfo.do`, `TaskSelectStdView.do`, `TaskStdDetail.do`는 404로 실패했다.
- `TaskViewStdPage.do`는 성공하더라도 KLAS 전체 HTML 템플릿을 반환하므로 본문 API로 사용하면 안 된다.

### 과제 상세 페이지 열기

```http
GET /std/lis/evltn/TaskViewStdPage.do
```

용도:

- 사용자가 KLAS 원문 페이지를 열 때만 사용한다.
- 앱 내 상세 본문은 `TaskStdView.do` 응답으로 렌더링한다.

성공 조건:

- HTTP status `2xx`
- 로그인 페이지가 아닌 HTML 페이지가 내려옴

실패/예외:

- 성공하더라도 앱 본문 렌더링 성공으로 간주하지 않는다.
- 로그인 페이지 HTML이 내려오면 세션 만료로 처리한다.
- 페이지 로드 실패 시 `TaskStdView.do` 상세 데이터가 있으면 네이티브 상세 화면은 유지한다.

### 과제 제출 페이지 열기

```http
GET /std/lis/evltn/TaskInsertStdPage.do
```

요청 파라미터:

```json
{
  "selectYearhakgi": "<yearHakgi>",
  "selectSubj": "<subject.value>",
  "ordseq": "<assignment.ordseq>",
  "weeklySeq": "<assignment.weeklySeq>",
  "weeklySubSeq": "<assignment.weeklySubSeq>"
}
```

응답 사용 방식:

- 응답은 HTML 제출 페이지다.
- 파일 업로드/제출 변경은 사용자 액션이므로 자동 동기화에서는 호출하지 않는다.
- 제출 상태 확인 용도로는 목록 응답과 상세 페이지를 우선 사용한다.

성공 조건:

- HTTP status `2xx`
- 로그인 페이지가 아닌 HTML 제출 페이지가 내려옴

실패/예외:

- 자동 동기화에서는 호출하지 않는다.
- 로그인 페이지 HTML이 내려오면 세션 만료로 처리한다.
- `ordseq`, `weeklySeq`, `weeklySubSeq`가 없으면 제출 페이지를 특정할 수 없으므로 원문 제출 열기를 비활성화한다.
- 페이지 로드 실패는 제출 상태 동기화 실패가 아니라 원문 제출 페이지 열기 실패로 분리해 표시한다.

추가 캡처가 필요한 항목:

- 지각 제출 가능 여부
- 첨부 파일 목록

앱 데이터 모델 초안:

```json
{
  "id": "klas-assignment-id",
  "subjectId": "subject.value",
  "subjectName": "수업명",
  "title": "과제명",
  "dueAt": "2026-06-01T23:59:59+09:00",
  "submitted": false,
  "detailUrl": "https://klas.kw.ac.kr/...",
  "raw": {}
}
```

미리알림 등록 규칙:

- 새 과제 발견 시 Reminder 생성
- 기존 과제의 마감 기한이 바뀌면 Reminder due date 갱신
- 제출 완료 과제는 설정에 따라 완료 처리 또는 유지
- Reminder 중복 방지를 위해 `assignment.id`와 Reminder identifier를 매핑해 저장

## 공지사항

### 강의 공지사항 페이지 열기

```http
GET /std/lis/sport/d052b8f845784c639f036b102fdc3023/BoardListStdPage.do
```

용도:

- KLAS 웹에서 "강의 공지사항" 메뉴가 여는 페이지다.
- 같은 게시판 계열로 "강의 묻고답하기", "강의 자료실", "수강생 자료실"도 UUID가 포함된 경로를 사용한다.

성공 조건:

- HTTP status `2xx`
- 로그인 페이지가 아닌 HTML 페이지가 내려옴

실패/예외:

- 로그인 페이지 HTML이 내려오면 세션 만료로 처리한다.
- 네이티브 공지 목록 데이터는 이 페이지가 아니라 `BoardStdList.do`로 판단한다.

확인된 게시판 페이지:

| 메뉴 | 경로 |
| --- | --- |
| 강의 공지사항 | `/std/lis/sport/d052b8f845784c639f036b102fdc3023/BoardListStdPage.do` |
| 강의 묻고답하기 | `/std/lis/sport/573f918c23984ae8a88c398051bb1263/BoardQnaListStdPage.do` |
| 강의 자료실 | `/std/lis/sport/6972896bfe72408eb72926780e85d041/BoardListStdPage.do` |
| 수강생 자료실 | `/std/lis/sport/70778131bf7a421aba99dded74b3fb6b/BoardListStdPage.do` |

### 강의 공지사항 목록 조회

```http
POST BoardStdList.do
```

공지사항 페이지 기준 상대 경로다. 절대 경로로는 아래처럼 호출된다.

```http
POST /std/lis/sport/d052b8f845784c639f036b102fdc3023/BoardStdList.do
```

요청 헤더: JSON 요청

요청 body:

```json
{
  "selectYearhakgi": "<yearHakgi>",
  "selectSubj": "<subject.value>",
  "currentPage": 0,
  "searchCondition": "ALL",
  "searchKeyword": null
}
```

확인된 응답 사용 방식:

- 실제 확인된 응답은 루트 객체다.
- `list`: 공지사항 목록
- `page`: 페이지네이션 정보
- `page` 필드는 `currentPage`, `pageSize`, `totalElements`, `totalPages`를 포함한다.

확인된 응답 구조:

```json
{
  "list": [
    {
      "rnum": 1,
      "boardNo": 1161280,
      "masterNo": 1000000,
      "grcode": "<grcode>",
      "year": "2026",
      "hakgi": "1",
      "subj": "<subject-id>",
      "bunban": "01",
      "categoryNm": null,
      "categoryId": null,
      "upperNo": 0,
      "refSort": 0,
      "sortOrdr": 0,
      "refLvl": 0,
      "title": "공지 제목",
      "topAt": "N",
      "othbcAt": "Y",
      "atchFileId": null,
      "readCnt": 0,
      "userNm": "작성자",
      "registerId": "<register-id>",
      "registDt": "2026-05-29T02:20:00.000+09:00",
      "myarticleAt": "N",
      "dateDiff": 0,
      "fileCnt": 0,
      "cmCnt": 0
    }
  ],
  "page": {
    "currentPage": 0,
    "pageSize": 10,
    "totalElements": 1,
    "totalPages": 1
  }
}
```

응답 사용 방식:

- `list`의 각 항목에서 `boardNo`를 꺼내 상세 페이지 이동에 사용한다.
- `masterNo`는 상세 조회(`BoardStdView.do`)에 필요하다. `boardNo`만 보내면 `board: null`이 내려오는 케이스를 확인했다.
- `page`는 전체 페이지 수, 현재 페이지, 다음/이전 표시 등 페이지네이션 UI에 사용한다.
- `title`은 공지 제목이다.
- `userNm`은 작성자 표시명이다.
- `registDt`는 작성일이며 `yyyy-MM-dd'T'HH:mm:ss.SSSXXX` 형식이다.
- `topAt`은 상단 고정/중요 공지 여부이며 `Y`면 중요 공지로 취급한다.
- `atchFileId`, `fileCnt`는 첨부 파일 표시와 다운로드 API 분석에 필요하다.
- `readCnt`, `cmCnt`, `dateDiff`는 조회수/댓글수/상대 경과일 표시에 쓸 수 있다.
- 앱은 공지별 안정 ID를 `board namespace + subject.value + boardNo` 조합으로 우선 구성할 수 있다.
- 현재 앱 구현은 루트 객체의 `list`와 위 실제 필드명만 기준으로 공지를 매핑한다.

성공 조건:

- 루트가 객체다.
- `list`가 배열이다.
- `page`가 없거나 일부 필드가 없어도 목록 표시 자체는 성공으로 처리 가능하다.
- `list`가 빈 배열이면 해당 과목 공지 없음 상태로 처리한다.

실패/예외:

- 공통 실패 응답 객체 또는 로그인 페이지 HTML이 내려오면 세션 만료/업무 오류로 처리한다.
- `list`가 없거나 배열이 아니면 공지 목록 구조 변경으로 보고 로드를 실패 처리한다.
- 공지 항목의 `boardNo` 또는 `masterNo`가 없으면 상세 조회가 불가능하므로 해당 항목은 목록에 표시하되 상세 진입을 비활성화하거나 제외한다.
- `registDt`가 파싱되지 않으면 작성일을 `확인 필요`로 표시하고 정렬에는 서버 순서를 유지한다.

### 강의 공지사항 상세 조회

```http
POST BoardStdView.do
```

공지사항 페이지 기준 상대 경로다. 절대 경로로는 아래처럼 호출된다.

```http
POST /std/lis/sport/d052b8f845784c639f036b102fdc3023/BoardStdView.do
```

요청 헤더: JSON 요청

요청 body:

```json
{
  "selectYearhakgi": "<yearHakgi>",
  "selectSubj": "<subject.value>",
  "boardNo": "<notice.boardNo>",
  "masterNo": "<notice.masterNo>",
  "cmd": "select"
}
```

확인된 응답 구조:

- 루트 객체는 `board`, `boardPre`, `boardNex`를 포함한다.
- `board`는 현재 공지 상세 정보다.
- `boardPre`, `boardNex`는 이전/다음 글 정보이며 없으면 `null`이다.
- 기존 `GET BoardViewStdPage.do`는 Angular 템플릿 HTML을 반환하므로 앱 상세 본문 렌더링에 직접 사용하지 않는다.

확인된 응답 예시:

```json
{
  "board": {
    "boardNo": 1161280,
    "masterNo": 1000000,
    "upperNo": 0,
    "refSort": 0,
    "sortOrdr": 0,
    "refLvl": 0,
    "title": "공지 제목",
    "content": "<p>공지 본문 HTML</p>",
    "topAt": "N",
    "othbcAt": "Y",
    "atchFileId": null,
    "readCnt": 0,
    "userNm": "작성자",
    "registerId": "<register-id>",
    "registerInfo": "...",
    "registerIp": "...",
    "registDt": "2026-05-29T02:20:00.000+09:00",
    "updusrId": null,
    "updusrInfo": null,
    "updusrIp": null,
    "updtDt": null,
    "categoryId": null,
    "langGbn": null,
    "myarticleAt": "N"
  },
  "boardPre": {
    "boardNo": 1161000,
    "masterNo": 1000000,
    "sortOrdr": 1,
    "title": "이전 글 제목"
  },
  "boardNex": null
}
```

응답 사용 방식:

- `board.title`은 상세 화면 제목이다.
- `board.content`는 공지 본문이며 HTML 문자열이므로 앱에서 텍스트로 정리해 표시한다.
- `board.userNm`은 작성자 표시명이다.
- `board.registDt`는 작성일이며 `yyyy-MM-dd'T'HH:mm:ss.SSSXXX` 형식이다.
- `board.readCnt`는 조회수다.
- `board.atchFileId`는 첨부 파일 API 분석에 필요한 파일 묶음 식별자다.
- `board.masterNo`, `board.boardNo`는 이전/다음 글 이동이나 원문 페이지 이동에 사용할 수 있다.

성공 조건:

- HTTP status `2xx`
- 루트 객체에 `board`가 있다.
- `board.title` 또는 `board.content` 중 하나 이상이 존재한다.

실패/예외:

- `masterNo` 없이 `boardNo`만 보내면 HTTP 200이어도 `board`, `boardPre`, `boardNex`가 모두 `null`인 응답을 확인했다.
- 잘못된 body 조합에서는 500 응답이 내려올 수 있다.
- `board`가 `null`이면 상세 로드 실패로 처리한다.
- `content`가 비어 있으면 본문 없음으로 표시하되 상세 로드 자체는 성공으로 둘 수 있다.
- 확인된 후보 중 `BoardViewStd.do`, `BoardStdInfo.do`, `BoardViewStdInfo.do`, `BoardSelectStdView.do`, `BoardStdDetail.do`는 404로 실패했다.
- `BoardViewStdPage.do`는 성공하더라도 KLAS 전체 HTML 템플릿을 반환하므로 본문 API로 사용하면 안 된다.

### 강의 공지사항 상세 페이지 열기

```http
GET BoardViewStdPage.do
```

용도:

- 사용자가 KLAS 원문 페이지를 열 때만 사용한다.
- 앱 내 상세 본문은 `BoardStdView.do` 응답으로 렌더링한다.

성공 조건:

- HTTP status `2xx`
- 로그인 페이지가 아닌 HTML 페이지가 내려옴

실패/예외:

- 성공하더라도 앱 본문 렌더링 성공으로 간주하지 않는다.
- 로그인 페이지 HTML이 내려오면 세션 만료로 처리한다.
- 페이지 로드 실패 시 `BoardStdView.do` 상세 데이터가 있으면 네이티브 상세 화면은 유지한다.

### 강의 일정 위젯 조회

```http
POST /std/cmn/frame/LctrumSchdulInfo.do
```

요청 헤더: JSON 요청

요청 body:

```json
{
  "selectGrcode": "<subject.grcode>",
  "selectYearhakgi": "<yearHakgi>",
  "selectSubj": "<subject.value>"
}
```

용도:

- KLAS 강의 페이지 상단의 강의 일정/강의실 정보 위젯을 조회한다.
- 공지사항 페이지 스크립트에서도 과목 변경 이벤트 시 호출된다.

응답 사용 방식:

- 실제 확인된 응답은 JSON이 아니라 문자열이다.
- 형식은 `요일 교시/강의실`이며, 복수 교시는 쉼표로 묶여 내려온다.
- 강의실이 미지정이면 `미지정` 문자열이 내려온다.
- 앱에서는 수업별 일정 요약 표시, 다음 강의 일정 표시, 과목 홈 화면 구성에 사용할 수 있다.

확인된 응답 구조:

```text
화 10,11교시/미지정
```

앱 처리:

- JSON으로 파싱하지 않는다.
- 시간표 그리드의 정규 수업 판단에는 `TimetableStdList.do`를 우선 사용한다.
- 이 응답은 과목 홈/요약 표시용 문자열로만 취급한다.

성공 조건:

- HTTP status `2xx`
- 응답이 문자열이다.
- 문자열이 비어 있지 않다.

실패/예외:

- JSON 공통 실패 응답 또는 로그인 페이지 HTML이 내려오면 세션 만료/업무 오류로 처리한다.
- 빈 문자열이면 일정 미등록 상태로 표시한다.
- `요일 교시/강의실` 형식으로 파싱되지 않아도 원문 문자열을 그대로 표시하고 시간표 계산에는 사용하지 않는다.

추가 캡처가 필요한 항목:

- 첨부 파일 다운로드 API
- 첨부 파일 목록

앱 데이터 모델 초안:

```json
{
  "id": "klas-notice-id",
  "subjectId": "subject.value",
  "subjectName": "수업명",
  "title": "공지 제목",
  "createdAt": "2026-05-28T10:00:00+09:00",
  "important": false,
  "detailUrl": "https://klas.kw.ac.kr/...",
  "raw": {}
}
```

## 시간표

### 시간표 페이지 열기

```http
GET /std/cps/atnlc/TimetableStdPage.do
```

용도:

- KLAS 웹에서 "수업시간표" 메뉴가 여는 페이지다.
- 페이지 스크립트는 수강 연도 목록과 시간표 목록 API를 호출한다.

성공 조건:

- HTTP status `2xx`
- 로그인 페이지가 아닌 HTML 페이지가 내려옴

실패/예외:

- 로그인 페이지 HTML이 내려오면 세션 만료로 처리한다.
- 네이티브 시간표 데이터는 이 페이지가 아니라 `TimetableStdList.do`로 판단한다.

### 수강 연도 목록 조회

```http
POST /std/cps/atnlc/AtnlcYearList.do
```

요청 헤더: JSON 요청

요청 body:

```json
{
  "searchYear": "2026",
  "searchHakgi": "1",
  "searchPgmNo": ""
}
```

확인된 응답 사용 방식:

- 페이지 스크립트는 응답을 `atnlcYearList`에 저장한다.
- 사용자는 이 목록을 통해 조회 가능한 연도/학기를 선택한다.
- 앱에서는 시간표 조회 필터의 연도/학기 선택지로 사용한다.

확인된 응답 구조:

```json
[
  {
    "year": "2026"
  },
  {
    "year": "2025"
  }
]
```

성공 조건:

- 루트가 배열이다.
- 각 항목은 최소 `year`를 포함한다.
- 배열이 비어 있으면 현재 선택 학기를 기본값으로 사용한다.

실패/예외:

- 공통 실패 응답 객체 또는 로그인 페이지 HTML이 내려오면 세션 만료/업무 오류로 처리한다.
- 루트가 배열이 아니면 시간표 필터 목록 로드를 실패 처리한다.
- `year`가 없는 항목은 제외한다.

### 시간표 목록 조회

```http
POST /std/cps/atnlc/TimetableStdList.do
```

요청 헤더: JSON 요청

요청 body:

```json
{
  "searchYear": "2026",
  "searchHakgi": "1",
  "searchPgmNo": ""
}
```

확인된 응답 사용 방식:

- 페이지 스크립트는 응답을 `timeTableList`에 저장한다.
- 시간표 셀 클릭 또는 과목 이동은 `setLctrumInfo('', yearhakgi, subj)` 후 `LctrumHomeStdPage.do`로 이동한다.
- 앱에서는 `timeTableList`를 요일/교시 단위 UI로 변환한다.
- 과목명, 강의실, 교수명, 교시 정보는 응답 필드명 확인 후 `TimetableEntry` 모델에 매핑한다.
- 실제 확인된 응답은 루트 배열이다.
- 각 배열 항목은 한 교시 행이며 `wtTime`이 교시를 나타낸다.
- `wtHasSchedule`이 `Y`인 행에 수업 정보가 들어 있다.
- 요일별 수업 정보는 suffix 숫자로 구분된다.
  - 월요일: `_1`
  - 화요일: `_2`
  - 수요일: `_3`
  - 목요일: `_4`
  - 금요일: `_5`
  - 토요일: `_6`
- 과목 ID: `wtSubj_<요일번호>`
- 과목명: `wtSubjNm_<요일번호>`
- 강의실: `wtLocHname_<요일번호>`
- 교수명: `wtProfNm_<요일번호>`
- 연속 교시 길이: `wtSpan_<요일번호>`
- 현재 앱 구현은 위 `wt*` 구조만 기준으로 시간표를 매핑한다.

확인된 응답 구조:

```json
[
  {
    "wtTime": "1",
    "wtHasSchedule": "Y",
    "wtSpan_2": "1",
    "wtYearhakgi_2": null,
    "wtSubj_2": "<subject-id>",
    "wtSubjNm_2": "수업명",
    "wtLocHname_2": "강의실",
    "wtProfNm_2": "교수명",
    "wtSubjPrintSeq_2": 1,
    "wtMoreBackSpan_2": "N",
    "wtMoreLine_2": 0,
    "wtSpan_5": "2",
    "wtYearhakgi_5": null,
    "wtSubj_5": "<subject-id>",
    "wtSubjNm_5": "수업명",
    "wtLocHname_5": "강의실",
    "wtProfNm_5": "교수명",
    "wtSubjPrintSeq_5": 1,
    "wtMoreBackSpan_5": "N",
    "wtMoreLine_5": 0
  }
]
```

온라인/미지정 수업 처리:

- `wtTime`이 `1`부터 `8`까지면 정규 교시로 표시한다.
- `wtTime > 8`이면 온라인 수업으로 표시한다.
- `wtLocHname_<요일번호>`가 비어 있거나 없으면 온라인 수업으로 표시한다.

성공 조건:

- 루트가 배열이다.
- 각 행은 `wtTime`을 포함한다.
- `wtHasSchedule === "Y"`인 행에서 요일 suffix별 `wtSubjNm_<n>` 또는 `wtSubj_<n>`를 읽을 수 있다.
- 배열이 비어 있으면 시간표 없음 상태로 표시한다.

실패/예외:

- 공통 실패 응답 객체 또는 로그인 페이지 HTML이 내려오면 세션 만료/업무 오류로 처리한다.
- 루트가 배열이 아니면 시간표 구조 변경으로 보고 로드를 실패 처리한다.
- `wtTime`이 없거나 숫자로 변환할 수 없는 행은 제외한다.
- `wtHasSchedule !== "Y"`인 행은 빈 교시로 처리하며 실패가 아니다.
- 요일 suffix의 과목명/과목 ID가 모두 없으면 해당 셀은 빈 셀로 처리한다.
- `wtSpan_<요일번호>`가 없거나 숫자로 변환할 수 없으면 기본값 `1`을 사용한다.
- `wtTime > 8` 또는 강의실 미지정 수업은 그리드에 억지로 배치하지 않고 온라인 수업 목록으로 분리 표시한다.

추가 캡처가 필요한 항목:

- `wtMoreBackSpan_<요일번호>`, `wtMoreLine_<요일번호>`, `wtSubjPrintSeq_<요일번호>`의 정확한 UI 의미

앱 데이터 모델 초안:

```json
{
  "subjectId": "subject.value",
  "subjectName": "수업명",
  "weekday": 1,
  "startTime": "09:00",
  "endTime": "10:15",
  "room": "새빛관 101",
  "professor": "교수명",
  "raw": {}
}
```

## KLAS Helper Extension 추가 조사 항목

아래 항목은 `kw-service/klas-helper-extension`의 `main` 브랜치에서 확인한 KLAS/Everytime 호출 중 기존 명세에 없던 항목이다. 실제 payload는 `fixtures/`의 대응 문서 기준으로 캡처한다.

### 팀프로젝트 목록

```http
POST /std/lis/evltn/PrjctStdList.do
GET /std/lis/evltn/PrjctStdPage.do
```

요청 body:

```json
{
  "selectSubj": "<subject-id>",
  "selectYearhakgi": "2026,1",
  "selectChangeYn": "Y"
}
```

확인된 사용 방식:

- extension 홈 화면은 수강 과목별로 `PrjctStdList.do`를 호출해 미제출 팀프로젝트 마감 정보를 계산한다.
- `PrjctStdPage.do`는 원문 페이지 이동용이다.
- 응답 shape는 과제 목록과 유사한 배열로 추정되며 `submityn`, `expiredate`, `reexpiredate`를 우선 확인한다.

성공 조건:

- HTTP status `2xx`
- 루트가 배열
- 빈 배열은 팀프로젝트 없음 상태

실패/예외:

- 공통 실패 응답 또는 로그인 HTML은 세션 만료/업무 오류로 처리한다.
- 배열이 아니면 구조 변경으로 진단한다.
- 마감 날짜를 파싱할 수 없는 항목은 알림 대상에서 제외한다.

### 상시퀴즈 목록

```http
POST /std/lis/evltn/AnytmQuizStdList.do
GET /std/lis/evltn/AnytmQuizStdPage.do
```

요청 body:

```json
{
  "selectSubj": "<subject-id>",
  "selectYearhakgi": "2026,1",
  "selectChangeYn": "Y"
}
```

확인된 사용 필드:

- `issubmit`: 제출 여부. `"Y"`면 제출 완료
- `edt`: 마감일
- `title`: 퀴즈 제목 후보

성공 조건:

- HTTP status `2xx`
- 루트가 배열
- 빈 배열은 퀴즈 없음 상태

실패/예외:

- `edt`가 없거나 날짜로 파싱할 수 없으면 마감 알림 대상에서 제외한다.
- `AnytmQuizStdPage.do`는 원문 페이지 이동용으로만 사용한다.

### 온라인 강의 학습 상태와 인증 체크

```http
POST /std/lis/evltn/SelectLrnSttusStd.do
POST /std/lis/evltn/CertiStdCheck.do
POST /std/lis/evltn/SaveLrnStatus.do
```

요청 body 후보:

```json
{
  "grcode": "<grcode>",
  "subj": "<subject-id>",
  "year": "2026",
  "hakgi": "1",
  "bunban": "01",
  "module": "01",
  "lesson": "001",
  "oid": "<oid>",
  "weeklyseq": "1",
  "weeklysubseq": "1"
}
```

`SelectLrnSttusStd.do` 응답 후보:

```text
Y
```

```text
N
```

확인된 사용 방식:

- extension은 온라인 강의 열기 전 `SelectLrnSttusStd.do`를 호출하고 응답이 `Y` 또는 `N`이면 뷰어 form을 제출한다.
- `CertiStdCheck.do`는 온라인 강의 인증 체크 로직에서 호출한다.
- `SaveLrnStatus.do`는 현재 Go 구현에 있는 학습 상태 저장 후보 API이며 응답 shape 추가 캡처가 필요하다.

성공 조건:

- `SelectLrnSttusStd.do`: HTTP `2xx`이고 body가 `Y` 또는 `N`
- `CertiStdCheck.do`: HTTP `2xx`이고 공통 실패 플래그가 없음
- `SaveLrnStatus.do`: HTTP `2xx`이고 공통 실패 플래그가 없음

실패/예외:

- 로그인 HTML은 세션 만료로 처리한다.
- `Y`/`N` 외 문자열은 구조 변경 또는 업무 상태 미확정으로 표시한다.

### 세션 연장

```http
POST /usr/cmn/login/UpdateSession.do
```

확인된 사용 방식:

- KLAS frame HTML 안에 정의된 세션 연장 함수가 이 API를 호출한다.
- extension은 5분마다 해당 함수를 찾아 실행한다.

성공 조건:

- HTTP status `2xx`
- 응답 body가 로그인 페이지 또는 공통 오류 페이지가 아님
- `Set-Cookie`가 있으면 저장된 쿠키를 갱신

실패/예외:

- 실패해도 즉시 로그아웃하지 않고 다음 인증 필요 API의 `loginRequired` 또는 로그인 HTML 판정으로 최종 세션 상태를 결정한다.
- 실제 요청 body와 성공 body shape는 추가 캡처가 필요하다.

### 강의계획서 필터 API

```http
POST /std/cps/atnlc/CmmnGamokList.do
POST /std/cps/atnlc/CmmnHakgwaList.do
POST /std/cps/atnlc/CmmnMagerCodeList.do
```

공통 과목 요청:

```json
{}
```

학과 요청:

```json
{
  "selectYear": "2026",
  "selecthakgi": "1"
}
```

전공 요청:

```json
{
  "selectYear": "2026",
  "selecthakgi": "1",
  "selecthakgwa": "<department-code>"
}
```

확인된 응답 후보:

```json
[
  {
    "code": "<code>",
    "codeName1": "공통 과목명"
  }
]
```

```json
[
  {
    "classCode": "<department-code>",
    "openMajorName": "학과명"
  }
]
```

성공 조건:

- HTTP status `2xx`
- 루트가 배열
- 빈 배열은 선택지 없음 상태

실패/예외:

- 공통 과목/전공은 `code`, `codeName1`이 없으면 해당 항목을 제외한다.
- 학과는 `classCode`, `openMajorName`이 없으면 해당 항목을 제외한다.

### 강의계획서 팝업과 수강인원 조회

```http
POST /std/cps/atnlc/CultureOptOneInfo.do
GET /std/cps/atnlc/popup/LectrePlanStdView.do
GET /std/cps/atnlc/popup/LectrePlanStdFixedView.do
GET /std/cps/atnlc/popup/LectrePlanStdNumPopup.do
POST /std/cps/atnlc/popup/LectrePlanStdCrtNum.do
```

`CultureOptOneInfo.do` 요청 후보:

```json
{
  "selectYear": "2026",
  "selecthakgi": "1",
  "selectSubj": "<subject-id>"
}
```

응답 후보:

```json
{
  "cultureOpt": false
}
```

`LectrePlanStdCrtNum.do` 요청 후보:

```json
{
  "currentNum": "0",
  "gwamokName": "수업명",
  "numText": "",
  "randomNum": "",
  "selectGrcode": "<grcode>",
  "selectSubj": "<subject-id>",
  "selectYear": "2026",
  "selectYearHakgi": "2026,1",
  "selecthakgi": "1",
  "stopFlag": ""
}
```

응답 후보:

```json
{
  "currentNum": "42"
}
```

처리 기준:

- `cultureOpt`가 truthy면 `LectrePlanStdFixedView.do`를 열고, 아니면 `LectrePlanStdView.do`를 연다.
- 팝업 HTML은 원문 표시용으로만 사용한다.
- `currentNum`이 숫자 문자열이면 수강인원 조회 성공으로 본다.

### 대학원 강의계획서

```http
GET /std/cps/atnlc/LectrePlanGdhlStdPage.do
POST /std/cps/atnlc/LectrePlanDaList.do
```

요청 body 후보:

```json
{
  "selectYear": "2026",
  "selecthakgi": "1",
  "selectGdhlitem": "<graduate-school-code>",
  "selectText": "과목명",
  "selectProfsr": "교수명"
}
```

확인된 사용 방식:

- extension은 `selectGdhlitem`이 없으면 검색하지 않는다.
- `LectrePlanDaList.do` 응답은 대학원 강의계획서 목록 배열로 사용한다.

성공 조건:

- HTTP status `2xx`
- 루트가 배열
- 빈 배열은 검색 결과 없음 상태

### 강의 묻고답하기 게시판

```http
POST /std/lis/sport/{questionBoardUid}/BoardStdList.do
GET /std/lis/sport/573f918c23984ae8a88c398051bb1263/BoardQnaListStdPage.do
```

요청 body:

```json
{
  "cmd": null,
  "pageInit": true,
  "selectYearhakgi": "2026,1",
  "selectSubj": "<subject-id>",
  "selectChangeYn": "Y",
  "searchCondition": "ALL",
  "searchKeyword": "",
  "currentPage": 0
}
```

확인된 응답 구조:

```json
{
  "list": [
    {
      "boardNo": "1",
      "masterNo": "10",
      "title": "질문 제목",
      "registDt": "2026-06-01 12:00:00",
      "userNm": "작성자"
    }
  ],
  "page": {
    "totalPages": "1",
    "totalElements": "1"
  }
}
```

처리 기준:

- `questionBoardUid`는 강의 홈 HTML의 `BoardQnaListStdPage.do` 링크에서 추출한다.
- `page.totalPages`가 있으면 전체 페이지를 순회할 수 있다.
- `registDt`가 24시간 이내인 항목은 신규 Q&A 후보로 표시할 수 있다.
- UID를 추출하지 못하면 Q&A 집계는 비활성화한다.

### KWCommons 슬라이드 리소스

```http
GET https://kwcommons.kw.ac.kr/viewer/ssplayer/uniplayer_support/content.php?content_id=<content-id>
GET <content_uri>/slide_list_<story-id>.xml
GET <content_uri>/<slide-image-uri>
```

`content.php` 응답에서 사용하는 XML 후보:

```xml
<content>
  <content_uri><![CDATA[https://kwcommons.kw.ac.kr/contents/<content-id>]]></content_uri>
  <story id="<story-id>"></story>
  <title><![CDATA[강의 제목]]></title>
</content>
```

slide list XML 후보:

```xml
<slide_list>
  <slide_image_src image_uri="slide/0001.jpg" />
</slide_list>
```

처리 기준:

- `content_uri`와 `story@id`가 모두 있으면 `slide_list_<story-id>.xml`을 조회한다.
- `slide_image_src@image_uri`는 `content_uri` 기준 상대 URL로 보정한다.
- 이미지 바이너리는 fixture에 저장하지 않고 metadata만 저장한다.
- 슬라이드 리소스가 없어도 동영상 다운로드는 실패로 처리하지 않는다.

### KLAS HTML page hook 경로

extension이 route로 감지하는 HTML 경로다. 앱에서는 원문 이동, 로그인 HTML 판정, fallback 검증에만 사용한다.

| 경로 | 용도 |
| --- | --- |
| `/std/cmn/frame/Frame.do` | 홈 화면, 세션 연장 함수, 시간표 widget |
| `/std/cps/atnlc/LectrePlanStdPage.do` | 학부 강의계획서 검색 화면 |
| `/std/cps/atnlc/popup/LectrePlanStdView.do` | 강의계획서 팝업 |
| `/std/cps/atnlc/popup/LectrePlanStdNumPopup.do` | 수강인원 조회 팝업 |
| `/std/cps/atnlc/LectrePlanGdhlStdPage.do` | 대학원 강의계획서 검색 화면 |
| `/std/cps/inqire/AtnlcScreStdPage.do` | 수강/성적 조회 화면 |
| `/std/cps/inqire/GradScreStdPage.do` | 졸업 성적 조회 화면 |
| `/std/cps/inqire/StandStdPage.do` | 석차 조회 화면 |
| `/std/cps/inqire/LctreEvlViewStdPage.do` | 강의평가 조회/작성 화면 |
| `/std/cps/inqire/LctreEvlStdPage.do` | 강의평가 이동 화면 |
| `/std/cps/inqire/ToeicStdPage.do` | 어학 성적 조회 화면 |
| `/std/lis/evltn/LctrumHomeStdPage.do` | 강의 홈 |
| `/std/lis/evltn/OnlineCntntsStdPage.do` | 온라인 강의 목록 화면 |
| `/spv/lis/lctre/viewer/LctreCntntsViewSpvPage.do` | 온라인 강의 viewer |
| `/std/cps/atnlc/TimetableStdPage.do` | 시간표 화면 |
| `/usr/cmn/login/LoginForm.do` | 로그인 화면 |
| `/std/ads/admst/MyInfoStdPage.do` | 내 정보 메뉴 링크 |
| `/std/ext/grdtn/GrdtnYnImprtyResnStdPage.do` | 졸업가부/졸업불가사유 메뉴 링크 |

처리 기준:

- HTML page fixture는 JSON API 대체 데이터로 사용하지 않는다.
- 로그인 화면 또는 공통 오류 화면이면 세션 만료/서버 오류로 분류한다.

## Everytime 강의평 API

Everytime API는 KLAS API가 아니며 KLAS 세션과 별개다. extension 저장소에는 수동 크롤러 스크립트와 packaged JSON loader가 있다. 자동 호출은 Everytime 로그인 세션, rate limit, 서비스 약관 리스크를 별도 검토하기 전까지 기본 비활성화한다.

### 강의 검색

```http
POST https://api.everytime.kr/find/lecture/list/keyword
Content-Type: application/x-www-form-urlencoded
Referrer: https://everytime.kr
```

요청 body:

```text
campusId=0&field=name&keyword=<lecture-name>&limit=20&offset=0
```

응답 후보:

```json
{
  "result": {
    "lectures": [
      {
        "id": 123456,
        "name": "강의명",
        "professor": "교수명",
        "rate": 4.5
      }
    ]
  }
}
```

성공 조건:

- HTTP status `2xx`
- `result.lectures`가 배열
- 빈 배열은 검색 결과 없음

실패/예외:

- 인증 실패, rate limit, HTML 오류 응답은 외부 서비스 오류로 분리한다.
- `name`, `professor`, `rate`가 없는 항목은 매칭 후보에서 제외한다.

### 강의 상세

```http
POST https://api.everytime.kr/find/lecture
Content-Type: application/x-www-form-urlencoded
Referrer: https://everytime.kr
```

요청 body:

```text
id=<everytime-lecture-id>
```

응답 후보:

```json
{
  "result": {
    "id": 123456,
    "name": "강의명",
    "professor": "교수명",
    "rate": {
      "average": 4.5,
      "count": 10,
      "items": []
    },
    "details": []
  }
}
```

성공 조건:

- HTTP status `2xx`
- `result`가 객체

실패/예외:

- `result`가 없으면 구조 변경 또는 삭제된 강의로 처리한다.
- 강의평 본문은 fixture에 원문 저장하지 않고 placeholder로 치환한다.

### Packaged JSON

```http
GET data/everytime-lectures.json
```

extension 내부 resource로 로드되는 정적 강의평 데이터다.

처리 기준:

- 앱에서 참고 데이터로 가져올 경우 외부 API가 아니라 bundled data import로 분류한다.
- 파일이 없거나 파싱 실패하면 강의평 기능만 비활성화하고 KLAS 기능은 유지한다.

## 주기적 동기화

동기화 대상:

- 수업 목록
- 강의 목록과 수강 가능 기간
- 과제 목록과 마감 기한
- 팀프로젝트 목록과 마감 기한
- 상시퀴즈 목록과 마감 기한
- 공지사항 목록
- 강의 묻고답하기 목록
- 시간표

권장 동작:

- 사용자가 설정한 주기로 백그라운드 동기화 실행
- 로그인 세션 만료 시 조용히 재로그인 시도
- 네트워크 실패 시 exponential backoff 적용
- 서버 오류가 반복되면 마지막 성공 데이터를 유지하고 사용자에게 상태 표시
- 새 과제 또는 새 공지사항 발견 시 macOS 알림 표시

## 추가 API 캡처 체크리스트

앱에서 직접 확인할 때는 로그인 후 `설정 > API 진단 > 응답 구조 확인`을 실행한다. 진단 결과는 응답 값이 아니라 JSON 루트 타입, 후보 배열 경로, 첫 번째 항목의 key 목록만 표시하므로 개인정보나 과제/공지 본문을 포함하지 않는다. `진단 결과 복사`로 복사한 key 목록을 아래 항목의 필드명 확정에 사용한다.

1. `TaskInsertStdPage.do`에서 과제 제출/수정 API 확인
2. 과제/공지 `atchFileId` 기반 첨부 파일 목록과 다운로드 API 확인
3. 온라인 강의 보기 버튼의 실제 `viewForm` 값 매핑과 뷰어 이동 경로 재확인
4. 세션 연장 API(`UpdateSession.do`)의 요청 body와 성공 body shape 확인
5. `PrjctStdList.do`, `AnytmQuizStdList.do`의 실제 응답 key 목록 확인
6. `SelectLrnSttusStd.do`, `CertiStdCheck.do`, `SaveLrnStatus.do`의 성공/실패 응답 형식 확인
7. 강의계획서 필터 API(`CmmnGamokList.do`, `CmmnHakgwaList.do`, `CmmnMagerCodeList.do`)의 빈 목록/세션 만료 응답 확인
8. `CultureOptOneInfo.do`, `LectrePlanStdCrtNum.do`, `LectrePlanDaList.do`의 실제 응답 shape 확인
9. 강의 Q&A `questionBoardUid` 추출 규칙과 `BoardStdList.do` pagination 응답 확인
10. KWCommons slide XML과 image URL 실패 응답 확인
11. Everytime API는 별도 로그인/약관/rate limit 검토 후 fixture 캡처 여부 결정

## 주의사항

- 이 API는 공식 문서가 아닌 리버스 엔지니어링 결과다.
- KLAS 엔드포인트와 파라미터는 예고 없이 변경될 수 있다.
- 자동 수강 기능은 학교 정책과 수업 운영 방식에 맞게 사용 여부를 별도 판단해야 한다.
- 개인정보, 학번, 비밀번호, 세션 쿠키, 다운로드한 강의 파일은 git에 포함하지 않는다.
