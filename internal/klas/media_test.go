package klas

import (
	"testing"
)

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
