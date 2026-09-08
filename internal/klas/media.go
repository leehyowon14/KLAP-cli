package klas

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type mediaCandidate struct {
	URL    string
	Target string
	Scope  string
}

func (c *Client) ResolveLectureMediaURL(ctx context.Context, contentID string) (string, error) {
	contentID = strings.TrimSpace(contentID)
	if contentID == "" {
		return "", errors.New("KWCommons 콘텐츠 ID가 없습니다")
	}

	endpoint := "https://kwcommons.kw.ac.kr/viewer/ssplayer/uniplayer_support/content.php?content_id=" + url.QueryEscape(contentID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "KLAP-CLI/0.1")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("KWCommons 요청 실패: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("KWCommons 응답 읽기 실패: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("KWCommons HTTP 오류: %s", response.Status)
	}

	mediaURL, err := ExtractMediaURL(body)
	if err != nil {
		return "", err
	}
	return mediaURL, nil
}

func ExtractKWCommonsContentID(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if id := extractKWCommonsContentIDFromURL(value); id != "" {
			return id
		}
	}
	return ""
}

func extractKWCommonsContentIDFromURL(value string) string {
	value = strings.TrimSpace(html.UnescapeString(value))
	if value == "" {
		return ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
		if id := extractKWCommonsContentIDFromPath(parsed.EscapedPath()); id != "" {
			return id
		}
		for _, key := range []string{"content_id", "contentId", "contentID", "contents"} {
			if id := normalizeKWCommonsContentID(parsed.Query().Get(key)); id != "" {
				return id
			}
		}
	}
	if id := extractKWCommonsContentIDAfterMarker(value, "em/"); id != "" {
		return id
	}
	for _, marker := range []string{"content_id=", "contentId=", "contentID=", "contents="} {
		if id := extractKWCommonsContentIDAfterMarker(value, marker); id != "" {
			return id
		}
	}
	return ""
}

func extractKWCommonsContentIDFromPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	for index, part := range parts {
		if part == "em" && index+1 < len(parts) {
			if id := normalizeKWCommonsContentID(parts[index+1]); id != "" {
				return id
			}
		}
	}
	return ""
}

func extractKWCommonsContentIDAfterMarker(value string, marker string) string {
	index := strings.Index(value, marker)
	if index < 0 {
		return ""
	}
	return normalizeKWCommonsContentID(value[index+len(marker):])
}

func normalizeKWCommonsContentID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, delimiter := range []string{"&", "?", "#", "/", "'", "\"", ")", " "} {
		if delimiterIndex := strings.Index(value, delimiter); delimiterIndex >= 0 {
			value = value[:delimiterIndex]
		}
	}
	if decoded, err := url.QueryUnescape(value); err == nil {
		value = decoded
	}
	return strings.TrimSpace(value)
}

func ExtractMediaURL(body []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	var stack []string
	var candidates []mediaCandidate
	var mainMedia string
	var allPrefix string

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			candidates = append(candidates, regexMediaCandidates(string(body))...)
			if mainMedia == "" {
				mainMedia = regexMainMedia(string(body))
			}
			break
		}
		switch typed := token.(type) {
		case xml.StartElement:
			stack = append(stack, typed.Name.Local)
			switch typed.Name.Local {
			case "media_uri":
				text, readErr := readElementText(decoder)
				if readErr != nil {
					continue
				}
				stack = stack[:len(stack)-1]
				candidate := mediaCandidate{URL: strings.TrimSpace(text), Scope: currentXMLScope(stack)}
				for _, attr := range typed.Attr {
					if attr.Name.Local == "target" {
						candidate.Target = strings.TrimSpace(attr.Value)
					}
				}
				if strings.Contains(candidate.URL, "[MEDIA_FILE]") {
					allPrefix = strings.Split(candidate.URL, "[MEDIA_FILE]")[0]
				}
				candidates = append(candidates, candidate)
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if mainMedia == "" {
		mainMedia = regexMainMedia(string(body))
	}

	for _, candidate := range candidates {
		if strings.Contains(candidate.URL, "[MEDIA_FILE]") && allPrefix == "" {
			allPrefix = strings.Split(candidate.URL, "[MEDIA_FILE]")[0]
		}
		if candidate.Scope == "desktop" && candidate.URL != "" && !strings.Contains(candidate.URL, "[MEDIA_FILE]") {
			return resolveKWCommonsURL(candidate.URL)
		}
	}
	for _, candidate := range candidates {
		if strings.EqualFold(candidate.Target, "all") && allPrefix != "" && mainMedia != "" {
			return resolveKWCommonsURL(allPrefix + mainMedia)
		}
	}
	for _, candidate := range candidates {
		if candidate.URL != "" && !strings.Contains(candidate.URL, "[MEDIA_FILE]") {
			return resolveKWCommonsURL(candidate.URL)
		}
	}

	return "", errors.New("KWCommons 응답에서 동영상 URL을 찾을 수 없습니다")
}

func normalizeKWCommonsPlayURL(value string, contentID string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if contentID != "" {
		return "https://kwcommons.kw.ac.kr/em/" + contentID
	}
	return value
}

func readElementText(decoder *xml.Decoder) (string, error) {
	var builder strings.Builder
	depth := 1
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			return "", err
		}
		switch typed := token.(type) {
		case xml.CharData:
			builder.Write([]byte(typed))
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		}
	}
	return builder.String(), nil
}

func currentXMLScope(stack []string) string {
	for index := len(stack) - 1; index >= 0; index-- {
		if stack[index] == "desktop" || stack[index] == "mobile" {
			return stack[index]
		}
	}
	return ""
}

func regexMediaCandidates(body string) []mediaCandidate {
	candidates := make([]mediaCandidate, 0)
	for _, match := range desktopMediaPattern.FindAllStringSubmatch(body, -1) {
		candidates = append(candidates, mediaCandidate{URL: strings.TrimSpace(match[1]), Scope: "desktop"})
	}
	for _, match := range mediaURIPattern.FindAllStringSubmatch(body, -1) {
		target := ""
		full := match[0]
		if strings.Contains(strings.ToLower(full), `target="all"`) || strings.Contains(strings.ToLower(full), `target='all'`) {
			target = "all"
		}
		candidates = append(candidates, mediaCandidate{URL: strings.TrimSpace(match[1]), Target: target})
	}
	return candidates
}

func regexMainMedia(body string) string {
	match := mainMediaPattern.FindStringSubmatch(body)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func resolveKWCommonsURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value, nil
	}
	base, err := url.Parse("https://kwcommons.kw.ac.kr/")
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(parsed).String(), nil
}
