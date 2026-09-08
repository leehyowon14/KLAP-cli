package klas

import (
	"testing"
)

func TestLooksLikeLoginHTML(t *testing.T) {
	cases := []struct {
		name string
		body []byte
		want bool
	}{
		{name: "html", body: []byte("<html><body>login</body></html>"), want: true},
		{name: "login form path", body: []byte("<script>location='/usr/cmn/login/LoginForm.do'</script>"), want: true},
		{name: "json", body: []byte(`{"loginRequired":true}`), want: false},
		{name: "empty", body: nil, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := looksLikeLoginHTML(tc.body); got != tc.want {
				t.Fatalf("looksLikeLoginHTML() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLooksLikeLoginPageHTML(t *testing.T) {
	if looksLikeLoginPageHTML([]byte("<html><body>viewer</body></html>")) {
		t.Fatal("looksLikeLoginPageHTML() should allow ordinary viewer html")
	}
	if !looksLikeLoginPageHTML([]byte("<html><script>location='/usr/cmn/login/LoginForm.do'</script></html>")) {
		t.Fatal("looksLikeLoginPageHTML() should detect login page")
	}
}

func TestFirstFieldError(t *testing.T) {
	got := firstFieldError([]fieldError{{Message: ""}, {Message: "개인번호 또는 비밀번호가 일치하지 않습니다."}}, "fallback")
	if got != "개인번호 또는 비밀번호가 일치하지 않습니다." {
		t.Fatalf("firstFieldError() = %q", got)
	}

	got = firstFieldError(nil, "fallback")
	if got != "fallback" {
		t.Fatalf("firstFieldError() fallback = %q", got)
	}
}

func TestSplitYearHakgi(t *testing.T) {
	year, hakgi := splitYearHakgi("2026,1")
	if year != "2026" || hakgi != "1" {
		t.Fatalf("splitYearHakgi() = %q, %q", year, hakgi)
	}

	year, hakgi = splitYearHakgi("2026-2")
	if year != "2026" || hakgi != "2" {
		t.Fatalf("splitYearHakgi() hyphen = %q, %q", year, hakgi)
	}
}

func TestNormalizeTermLabelSeasonSemesters(t *testing.T) {
	if got := normalizeTermLabel("2026년도 3학기", "2026,3"); got != "2026년도 여름학기" {
		t.Fatalf("normalizeTermLabel() summer = %q", got)
	}
	if got := normalizeTermLabel("", "2026,4"); got != "2026년도 겨울학기" {
		t.Fatalf("normalizeTermLabel() winter = %q", got)
	}
	if got := normalizeTermLabel("2026년도 1학기", "2026,1"); got != "2026년도 1학기" {
		t.Fatalf("normalizeTermLabel() regular = %q", got)
	}
}

func TestExtractKWCommonsContentID(t *testing.T) {
	got := ExtractKWCommonsContentID("https://kwcommons.kw.ac.kr/em/content-123&contents=abc", "")
	if got != "content-123" {
		t.Fatalf("ExtractKWCommonsContentID() = %q", got)
	}

	got = ExtractKWCommonsContentID("", "https://kwcommons.kw.ac.kr/em/fallback-456")
	if got != "fallback-456" {
		t.Fatalf("ExtractKWCommonsContentID() fallback = %q", got)
	}

	cases := map[string]string{
		"https://kwcommons.kw.ac.kr/viewer/ssplayer/uniplayer_support/content.php?content_id=viewer-789": "viewer-789",
		"https://kwcommons.kw.ac.kr/em/path-123/?contents=ignored":                                       "path-123",
		"https://kwcommons.kw.ac.kr/em/html-123&amp;contents=abc":                                        "html-123",
		"javascript:openPlayer('https://kwcommons.kw.ac.kr/em/script-123?contents=abc')":                 "script-123",
		"https://kwcommons.kw.ac.kr/player?contents=query-123":                                           "query-123",
	}
	for value, want := range cases {
		if got := ExtractKWCommonsContentID(value); got != want {
			t.Fatalf("ExtractKWCommonsContentID(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestExtractMediaURLDesktop(t *testing.T) {
	body := []byte(`<content><desktop><media_uri>https://media.example.com/video.mp4</media_uri></desktop></content>`)

	got, err := ExtractMediaURL(body)
	if err != nil {
		t.Fatalf("ExtractMediaURL() error = %v", err)
	}
	if got != "https://media.example.com/video.mp4" {
		t.Fatalf("ExtractMediaURL() = %q", got)
	}
}

func TestExtractMediaURLPreservesExternalDesktopHost(t *testing.T) {
	body := []byte(`<content><desktop><media_uri>https://professor-media.example.edu/lecture/path/video.m3u8?token=signed</media_uri></desktop></content>`)

	got, err := ExtractMediaURL(body)
	if err != nil {
		t.Fatalf("ExtractMediaURL() error = %v", err)
	}
	want := "https://professor-media.example.edu/lecture/path/video.m3u8?token=signed"
	if got != want {
		t.Fatalf("ExtractMediaURL() = %q, want %q", got, want)
	}
}

func TestExtractMediaURLNestedMainMediaDesktop(t *testing.T) {
	body := []byte(`<?xml version="1.0"?>
<content version="1.0"><content_playing_info version="1.0"><content_id>699bd27c797cb</content_id><main_media><desktop><html5><method>progressive</method><media_uri>https://kwcommons.kw.ac.kr/contents5/KW10000001/699bd27c797cb/contents/media_files/mobile/ssmovie.mp4</media_uri></html5><flash_fallback><method>pseudo</method><media_uri>https://kwcommons.kw.ac.kr/contents5_pseudo/KW10000001/699bd27c797cb/contents/media_files/mobile/ssmovie.mp4</media_uri></flash_fallback></desktop><mobile><html5><method>progressive</method><media_uri>https://kwcommons.kw.ac.kr/contents5/KW10000001/699bd27c797cb/contents/media_files/mobile/ssmovie.mp4</media_uri></html5></mobile></main_media></content_playing_info></content>`)

	got, err := ExtractMediaURL(body)
	if err != nil {
		t.Fatalf("ExtractMediaURL() error = %v", err)
	}
	want := "https://kwcommons.kw.ac.kr/contents5/KW10000001/699bd27c797cb/contents/media_files/mobile/ssmovie.mp4"
	if got != want {
		t.Fatalf("ExtractMediaURL() = %q, want %q", got, want)
	}
}

func TestExtractMediaURLFallbackMainMedia(t *testing.T) {
	body := []byte(`<content><media_uri target="all">https://media.example.com/path/[MEDIA_FILE]</media_uri><main_media media_id="m1">video.mp4</main_media></content>`)

	got, err := ExtractMediaURL(body)
	if err != nil {
		t.Fatalf("ExtractMediaURL() error = %v", err)
	}
	if got != "https://media.example.com/path/video.mp4" {
		t.Fatalf("ExtractMediaURL() = %q", got)
	}
}

func TestHTMLToTextKeepsBlockBreaks(t *testing.T) {
	input := `<p>과제 설명</p><p><a href="https://example.com">https://example.com</a></p><p>제출 내용</p><ol><li>GitHub repository 주소</li><li>youtube 링크</li></ol>`

	got := htmlToText(input)
	want := "과제 설명\n\nhttps://example.com\n\n제출 내용\n\n- GitHub repository 주소\n- youtube 링크"
	if got != want {
		t.Fatalf("htmlToText() = %q, want %q", got, want)
	}
}

func TestHTMLToTextKeepsNumberedListCompact(t *testing.T) {
	input := `<p>제출 내용</p><p>1. GitHub repository 주소</p><p>2. youtube 링크</p>`

	got := htmlToText(input)
	want := "제출 내용\n\n1. GitHub repository 주소\n2. youtube 링크"
	if got != want {
		t.Fatalf("htmlToText() = %q, want %q", got, want)
	}
}
