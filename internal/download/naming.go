package download

import (
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"net/url"
	"path/filepath"
	"strings"
)

func (*Client) VideoPath(root string, courseName string, lecture app.LectureFileName, mediaURL string, weekOrder int) string {
	return filepath.Join(root, lectureCourseDirName(courseName), "video", lectureFilename(lecture, mediaURL, weekOrder))
}

func lectureCourseDirName(courseName string) string {
	courseDir := sanitizePathComponent(courseName)
	if courseDir == "" {
		return "course"
	}
	return courseDir
}

func lectureFilename(lecture app.LectureFileName, mediaURL string, weekOrder int) string {
	extension := ".mp4"
	if parsed, err := url.Parse(mediaURL); err == nil {
		if ext := filepath.Ext(parsed.Path); ext != "" {
			extension = ext
		}
	}

	if week := lecture.Week; week > 0 && weekOrder > 0 {
		title := sanitizePathComponent(firstNonEmpty(lecture.Title, lecture.ModuleTitle, lecture.ContentID, lecture.LearningSeq, "lecture"))
		return fmt.Sprintf("%d-%d. %s%s", week, weekOrder, title, extension)
	}

	parts := []string{
		sanitizePathComponent(lecture.ModuleTitle),
		sanitizePathComponent(lecture.Title),
	}

	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		filtered = append(filtered, "lecture")
	}
	return strings.Join(filtered, "_") + extension
}

func sanitizePathComponent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\n", " ",
		"\r", " ",
		"\t", " ",
	)
	value = replacer.Replace(value)
	value = strings.Join(strings.Fields(value), " ")
	if len([]rune(value)) > 120 {
		runes := []rune(value)
		value = string(runes[:120])
	}
	return strings.Trim(value, ". ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
