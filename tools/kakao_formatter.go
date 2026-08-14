package tools

import (
	"fmt"
	"html"
	"strings"
)

// formatKakaoDocuments converts a normalized Kakao search response into text
// suitable for an MCP TextContent response. It has no HTTP or configuration
// dependency, so output changes can be tested independently.
func formatKakaoDocuments(collection string, docs []map[string]any) string {
	var sb strings.Builder
	for _, d := range docs {
		title := stripHTMLTags(kakaoString(d, "title"))
		if title == "" {
			title = kakaoString(d, "name")
		}
		link := kakaoString(d, "url")
		if link == "" {
			link = kakaoString(d, "doc_url")
		}
		contents := stripHTMLTags(kakaoString(d, "contents"))
		fmt.Fprintf(&sb, "제목: %s\n링크: %s\n설명: %s", title, link, contents)
		switch collection {
		case "blog":
			fmt.Fprintf(&sb, "\n블로그: %s", kakaoString(d, "blogname"))
		case "cafe":
			fmt.Fprintf(&sb, "\n카페: %s", kakaoString(d, "cafename"))
		case "book":
			fmt.Fprintf(&sb, "\n저자: %s\n출판사: %s\nISBN: %s", kakaoString(d, "authors"), kakaoString(d, "publisher"), kakaoString(d, "isbn"))
		case "image":
			fmt.Fprintf(&sb, "\n이미지: %s\n출처: %s", kakaoString(d, "image_url"), kakaoString(d, "display_sitename"))
		case "vclip":
			fmt.Fprintf(&sb, "\n업로더: %s\n썸네일: %s", kakaoString(d, "author"), kakaoString(d, "thumbnail"))
		}
		sb.WriteString("\n---\n")
	}
	return sb.String()
}

func formatKakaoPlaces(docs []map[string]any) string {
	var sb strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&sb, "장소: %s\n카테고리: %s\n전화: %s\n주소: %s\n도로명: %s\n좌표: %s, %s\n링크: %s\n---\n", kakaoString(d, "place_name"), kakaoString(d, "category_name"), kakaoString(d, "phone"), kakaoString(d, "address_name"), kakaoString(d, "road_address_name"), kakaoString(d, "x"), kakaoString(d, "y"), kakaoString(d, "place_url"))
	}
	return sb.String()
}

func stripHTMLTags(s string) string {
	s = strings.ReplaceAll(s, "<b>", "")
	s = strings.ReplaceAll(s, "</b>", "")
	return html.UnescapeString(s)
}

func kakaoString(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return html.UnescapeString(s)
	}
	if a, ok := m[key].([]any); ok {
		var values []string
		for _, v := range a {
			if s, ok := v.(string); ok {
				values = append(values, s)
			}
		}
		return strings.Join(values, ", ")
	}
	return ""
}
