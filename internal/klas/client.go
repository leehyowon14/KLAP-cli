package klas

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://klas.kw.ac.kr"

type Client struct {
	httpClient *http.Client
	jar        http.CookieJar
	baseURL    *url.URL
}

func NewClient() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("cookie jar 생성 실패: %w", err)
	}

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 20 * time.Second,
		},
		jar:     jar,
		baseURL: parsedBaseURL,
	}, nil
}

func mapString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(values[key]))
}

func parseKoreanDateTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	location, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		location = time.FixedZone("KST", 9*60*60)
	}

	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04"} {
		parsed, err := time.ParseInLocation(layout, value, location)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

func parseKlasDateTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return &parsed
	}
	return parseKoreanDateTime(value)
}

var (
	htmlBreakPattern      = regexp.MustCompile(`(?i)<br\s*/?>`)
	htmlBlockClosePattern = regexp.MustCompile(`(?i)</(p|div|section|article|header|footer|h[1-6]|li|ul|ol|table|thead|tbody|tr)>`)
	htmlBlockOpenPattern  = regexp.MustCompile(`(?i)<(p|div|section|article|header|footer|h[1-6]|ul|ol|table|thead|tbody|tr)(\s+[^>]*)?>`)
	htmlListItemPattern   = regexp.MustCompile(`(?i)<li(\s+[^>]*)?>`)
	htmlCellPattern       = regexp.MustCompile(`(?i)</t[dh]>\s*<t[dh](\s+[^>]*)?>`)
	htmlTagPattern        = regexp.MustCompile(`<[^>]+>`)
	htmlBlankLinePattern  = regexp.MustCompile(`\n{3,}`)
	htmlListGapPattern    = regexp.MustCompile(`(?m)(- [^\n]+)\n\n- `)
	htmlNumberGapPattern  = regexp.MustCompile(`(?m)([0-9]+\. [^\n]+)\n\n([0-9]+\. )`)
	htmlLineSpacePattern  = regexp.MustCompile(`[ \t]+\n`)
	mediaURIPattern       = regexp.MustCompile(`(?is)<media_uri(?:\s+[^>]*)?>([^<]+)</media_uri>`)
	desktopMediaPattern   = regexp.MustCompile(`(?is)<desktop\b[^>]*>.*?<media_uri(?:\s+[^>]*)?>([^<]+)</media_uri>.*?</desktop>`)
	mainMediaPattern      = regexp.MustCompile(`(?is)<main_media(?:\s+[^>]*)?>([^<]+)</main_media>`)
	lectureKeyPattern     = regexp.MustCompile(`["']lecKey["']\s*:\s*['"]([^'"]+)['"]`)
)

func htmlToText(value string) string {
	value = htmlBreakPattern.ReplaceAllString(value, "\n")
	value = htmlCellPattern.ReplaceAllString(value, " ")
	value = htmlListItemPattern.ReplaceAllString(value, "\n- ")
	value = htmlBlockOpenPattern.ReplaceAllString(value, "\n")
	value = htmlBlockClosePattern.ReplaceAllString(value, "\n")
	value = htmlTagPattern.ReplaceAllString(value, "")
	value = decodeHTMLEntities(value)
	value = htmlLineSpacePattern.ReplaceAllString(value, "\n")
	value = htmlBlankLinePattern.ReplaceAllString(value, "\n\n")
	for htmlListGapPattern.MatchString(value) {
		value = htmlListGapPattern.ReplaceAllString(value, "$1\n- ")
	}
	for htmlNumberGapPattern.MatchString(value) {
		value = htmlNumberGapPattern.ReplaceAllString(value, "$1\n$2")
	}
	return strings.TrimSpace(value)
}

func decodeHTMLEntities(value string) string {
	replacements := map[string]string{
		"&nbsp;": " ",
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": `"`,
		"&#39;":  "'",
		"&apos;": "'",
	}
	for entity, replacement := range replacements {
		value = strings.ReplaceAll(value, entity, replacement)
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func rowString(row map[string]any, key string) string {
	value, ok := row[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return strings.TrimSpace(typed.String())
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(typed, 'f', -1, 64))
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
