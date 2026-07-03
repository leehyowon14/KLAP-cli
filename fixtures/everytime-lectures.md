# Everytime Lectures Fixture

## Endpoints

```http
POST https://api.everytime.kr/find/lecture/list/keyword
POST https://api.everytime.kr/find/lecture
GET data/everytime-lectures.json
```

## Request

강의 검색:

```http
Content-Type: application/x-www-form-urlencoded
Referrer: https://everytime.kr
```

```text
campusId=0&field=name&keyword=<lecture-name>&limit=20&offset=0
```

강의 상세:

```text
id=<everytime-lecture-id>
```

## Success Fixtures

- `payloads/everytime-search.success.json`
- `payloads/everytime-search.empty.json`
- `payloads/everytime-search.multi-page.json`
- `payloads/everytime-lecture.success.json`
- `payloads/everytime-lecture.no-rate.json`
- `payloads/everytime-packaged-data.success.json`

필수 후보 구조:

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

## Failure Fixtures

- `payloads/everytime-search.unauthorized.json`
- `payloads/everytime-search.rate-limited.json`
- `payloads/everytime-search.http-error.html`
- `payloads/everytime-lecture.missing-result.json`
- `payloads/everytime-packaged-data.missing.json`

## Parser Assertions

- Everytime API는 KLAS 세션과 별개이며 credentials가 필요한 외부 서비스로 분류한다.
- 검색 결과에서 `name`, `professor`, `rate`가 없는 항목은 강의평 매칭 후보에서 제외한다.
- 상세 응답은 `result` 객체가 있어야 한다.
- packaged JSON은 extension resource fixture로만 사용하고 런타임 네트워크 API로 취급하지 않는다.
- API 호출 자동화는 rate limit과 서비스 약관 리스크를 별도 검토하기 전까지 기본 비활성화한다.

## Redaction Rules

- 강의평 본문은 원문 저장하지 않고 짧은 placeholder로 치환한다.
- Everytime 사용자 식별자, 쿠키, 세션성 header는 저장하지 않는다.
