package tools

import (
	"fmt"
	"html"
	"strings"
)

func formatKakaoWeb(docs []kakaoWebDocument) string {
	var sb strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&sb, "제목: %s\n링크: %s\n설명: %s\n---\n", clean(d.Title), d.URL, clean(d.Contents))
	}
	return sb.String()
}
func formatKakaoBlog(docs []kakaoBlogDocument) string {
	var sb strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&sb, "제목: %s\n링크: %s\n설명: %s\n블로그: %s\n---\n", clean(d.Title), d.URL, clean(d.Contents), d.BlogName)
	}
	return sb.String()
}
func formatKakaoCafe(docs []kakaoCafeDocument) string {
	var sb strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&sb, "제목: %s\n링크: %s\n설명: %s\n카페: %s\n---\n", clean(d.Title), d.URL, clean(d.Contents), d.CafeName)
	}
	return sb.String()
}
func formatKakaoBook(docs []kakaoBookDocument) string {
	var sb strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&sb, "제목: %s\n링크: %s\n설명: %s\n저자: %s\n출판사: %s\nISBN: %s\n---\n", clean(d.Title), d.URL, clean(d.Contents), strings.Join(d.Authors, ", "), d.Publisher, d.ISBN)
	}
	return sb.String()
}
func formatKakaoImage(docs []kakaoImageDocument) string {
	var sb strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&sb, "이미지: %s\n썸네일: %s\n출처: %s\n링크: %s\n---\n", d.ImageURL, d.ThumbnailURL, d.DisplaySiteName, d.DocURL)
	}
	return sb.String()
}
func formatKakaoVideo(docs []kakaoVideoDocument) string {
	var sb strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&sb, "제목: %s\n링크: %s\n업로더: %s\n썸네일: %s\n재생시간: %d초\n---\n", clean(d.Title), d.URL, d.Author, d.Thumbnail, d.PlayTime)
	}
	return sb.String()
}
func formatKakaoPlaces(docs []kakaoPlaceDocument) string {
	var sb strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&sb, "장소: %s\n카테고리: %s\n전화: %s\n주소: %s\n도로명: %s\n좌표: %s, %s\n거리: %s\n링크: %s\n---\n", d.PlaceName, d.CategoryName, d.Phone, d.AddressName, d.RoadAddressName, d.X, d.Y, d.Distance, d.PlaceURL)
	}
	return sb.String()
}
func clean(s string) string {
	return html.UnescapeString(strings.ReplaceAll(strings.ReplaceAll(s, "<b>", ""), "</b>", ""))
}
