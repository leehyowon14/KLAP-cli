package klas

import "testing"

func TestNativeViewerContentID(t *testing.T) {
	for _, html := range []string{`$('#learnEmbed').html('<EMBED src="https://kwcommons.kw.ac.kr/em/example123">')`, `window.frames["LEARN_IFRAME"].location.href="https://kwcommons.kw.ac.kr/em/example123";`} {
		id, e := nativeViewerContentID([]byte(html))
		if e != nil || id != "example123" {
			t.Fatalf("%s %v", id, e)
		}
	}
	for _, html := range []string{"", `<iframe src="default.htm"></iframe>`, `<html>로그인이 필요합니다</html>`} {
		if _, e := nativeViewerContentID([]byte(html)); e == nil {
			t.Fatal("missing media accepted")
		}
	}
}
