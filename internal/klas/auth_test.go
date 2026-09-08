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
