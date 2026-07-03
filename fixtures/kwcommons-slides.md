# KWCommons Slides Fixture

## Endpoint

```http
GET https://kwcommons.kw.ac.kr/viewer/ssplayer/uniplayer_support/content.php?content_id=<content-id>
GET <content_uri>/slide_list_<story-id>.xml
GET <content_uri>/<slide-image-uri>
```

## Request

`content.php` 응답 XML에서 `content_uri`, `story@id`, `slide_image_src@image_uri`를 순서대로 추출한다.

## Success Fixtures

- `payloads/kwcommons-content.with-slides.xml`
- `payloads/kwcommons-content.video-only.xml`
- `payloads/kwcommons-slide-list.success.xml`
- `payloads/kwcommons-slide-list.empty.xml`
- `payloads/kwcommons-slide-image.success.meta.json`

필수 XML 후보:

```xml
<content>
  <content_uri><![CDATA[https://kwcommons.kw.ac.kr/contents/<content-id>]]></content_uri>
  <story id="<story-id>"></story>
  <title><![CDATA[강의 제목]]></title>
</content>
```

```xml
<slide_list>
  <slide_image_src image_uri="slide/0001.jpg" />
</slide_list>
```

## Failure Fixtures

- `payloads/kwcommons-content.missing-content-uri.xml`
- `payloads/kwcommons-content.missing-story.xml`
- `payloads/kwcommons-slide-list.http-error.html`
- `payloads/kwcommons-slide-image.http-error.html`

## Parser Assertions

- `content_uri`가 없으면 슬라이드 다운로드를 비활성화한다.
- `story`가 없으면 동영상만 다운로드 가능 상태로 둔다.
- slide image URL은 `content_uri` 기준 상대 경로로 보정한다.
- 이미지 payload는 git에 넣지 않고 metadata fixture만 둔다.

## Redaction Rules

- 콘텐츠 ID, 제목, 내부 경로의 계정성 값은 치환한다.
- XML tag 구조와 CDATA 여부는 유지한다.
