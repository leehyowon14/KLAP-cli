# Lecture Download Fixtures

## Scope

온라인 강의 목록의 `mvpLink`/`starting`에서 KWCommons 콘텐츠 ID를 추출하고 실제 동영상 URL을 얻는 흐름이다.

## Inputs

- `payloads/lectures-list.success.json`
- KWCommons HTML 또는 API 응답 fixture

## Success Fixtures

- `payloads/lecture-download.kwcommons-html.html`
- `payloads/lecture-download.media-url.json`
- `payloads/lecture-download.desktop-media-uri.html`
- `payloads/lecture-download.fallback-media-file.html`
- `payloads/lecture-download.external-host-media-uri.html`

## Observed Path Variants

### 외부/교수자 미디어 host 절대 URL

일부 강의는 일반 KWCommons 저장소 경로와 다르게 교수자/외부 서버의 직접 미디어 경로가
`<media_uri>`에 들어오는 케이스가 있다. 2025학년도 1학기 `사회과학교양세미나` 계열에서
이 변형을 관찰했지만, 과목명에 종속된 예외가 아니라 모든 강의 다운로드에서 동일하게 처리한다.

대표 구조:

```xml
<content>
  <desktop>
    <media_uri>https://<external-media-host>/<course-or-owner-path>/<video-file>.mp4</media_uri>
  </desktop>
</content>
```

파서 기대값:

- `content.php?content_id=<content-id>` 조회 흐름은 동일하게 유지한다.
- `<desktop>` 내부 `<media_uri>`가 `kwcommons.kw.ac.kr`가 아니어도 절대 URL이면 그대로 최종 다운로드 URL로 사용한다.
- host가 다르다는 이유만으로 KWCommons base URL을 붙이지 않는다.
- 외부 host URL에 query string이나 signed token이 있으면 그대로 보존한다.
- 파일 확장자는 `.mp4` 외에도 서버가 내려준 path component의 확장자를 유지한다.

## Failure Fixtures

- `payloads/lecture-download.missing-content-id.json`
- `payloads/lecture-download.no-media-url.html`
- `payloads/lecture-download.expired.html`

## Parser Assertions

- `https://kwcommons.kw.ac.kr/em/<content-id>`에서 `<content-id>`를 추출한다.
- `mvpLink`가 없으면 `starting`을 사용한다.
- 동영상 URL 후보가 여러 개면 desktop media URI를 우선한다.
- desktop media URI가 외부 host의 절대 URL이면 그대로 보존한다.
- 외부 host 절대 URL은 과목명과 무관하게 모든 강의에 적용한다.
- URL이 없으면 다운로드 불가 상태로 둔다.
- 파일명은 과목명/강의명/path component를 모두 sanitize한다.

## Redaction Rules

- 콘텐츠 ID와 media URL path token은 치환한다.
- 확장자, host, query 구조는 유지한다.
