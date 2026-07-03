# Lecture Download Fixtures

## Scope

온라인 강의 목록의 `mvpLink`/`starting`에서 KWCommons 콘텐츠 ID를 추출하고 실제 동영상 URL을 얻는 흐름이다.

## Endpoints

```http
GET https://kwcommons.kw.ac.kr/viewer/ssplayer/uniplayer_support/content.php?content_id=<content-id>
GET <media_uri>
```

## Inputs

- `payloads/lectures-list.success.json`
- KWCommons `content.php` XML 응답
- media URL HEAD/metadata fixture

## Success Fixtures

- `payloads/lecture-download.content-progressive.xml`
- `payloads/lecture-download.content-pseudo.xml`
- `payloads/lecture-download.content-multi-media.xml`
- `payloads/lecture-download.content-missing-media.xml`
- `payloads/lecture-download.media-head.success.meta.json`
- `payloads/lecture-download.external-host-media-uri.xml`

필수 XML 후보:

```xml
<content>
  <content_playing_info>
    <main_media>
      <desktop>
        <html5>
          <method>progressive</method>
          <media_uri>https://kwcommons.kw.ac.kr/contents/.../ssmovie.mp4</media_uri>
        </html5>
      </desktop>
    </main_media>
  </content_playing_info>
</content>
```

## Observed Path Variants

### KWCommons progressive media

`desktop/html5/media_uri`가 있으면 이 값을 최종 다운로드 URL로 사용한다.

### KWCommons pseudo fallback media

`desktop/html5/media_uri`가 없고 `desktop/flash_fallback/media_uri`가 있으면 fallback 값을 사용한다.

### 외부/교수자 미디어 host 절대 URL

일부 강의는 일반 KWCommons 저장소 경로와 다르게 교수자/외부 서버의 직접 미디어 경로가
`<media_uri>`에 들어오는 케이스가 있다. 과목명에 종속된 예외가 아니라 모든 강의 다운로드에서 동일하게 처리한다.

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

- `payloads/lecture-download.content-http-error.html`
- `payloads/lecture-download.content-malformed.xml`
- `payloads/lecture-download.media-403.meta.json`
- `payloads/lecture-download.media-timeout.meta.json`
- `payloads/lecture-download.missing-content-id.json`
- `payloads/lecture-download.no-media-url.xml`

## Parser Assertions

- `https://kwcommons.kw.ac.kr/em/<content-id>`에서 `<content-id>`를 추출한다.
- `mvpLink`가 없으면 `starting`을 사용한다.
- `desktop/html5/media_uri`를 우선 사용한다.
- `desktop/flash_fallback/media_uri`는 html5 URI가 없을 때만 사용한다.
- desktop media URI가 외부 host의 절대 URL이면 그대로 보존한다.
- 외부 host 절대 URL은 과목명과 무관하게 모든 강의에 적용한다.
- `media_uri`가 없으면 다운로드 불가로 표시한다.
- 파일명은 과목명/강의명/path component를 모두 sanitize한다.
- 실제 mp4 payload는 저장하지 않고 HEAD/metadata fixture만 보관한다.

## Redaction Rules

- 콘텐츠 ID와 내부 KWCommons 경로의 계정성 값은 치환한다.
- 외부 host 절대 URL의 host, 확장자, query 구조는 유지한다.
