package tools

import (
	"strings"
	"testing"
)

func TestFormatKakaoDocumentsByCollection(t *testing.T) {
	cases := []struct{ name, got, want string }{
		{"web", formatKakaoWeb([]kakaoWebDocument{{Title: "<b>제목</b>", Contents: "A&amp;B", URL: "https://example.com"}}), "A&B"},
		{"blog", formatKakaoBlog([]kakaoBlogDocument{{Title: "제목", BlogName: "테스트 블로그"}}), "테스트 블로그"},
		{"book", formatKakaoBook([]kakaoBookDocument{{Title: "책", Authors: []string{"홍길동"}, Publisher: "출판사"}}), "홍길동"},
		{"image", formatKakaoImage([]kakaoImageDocument{{ImageURL: "https://image.example"}}), "https://image.example"},
		{"video", formatKakaoVideo([]kakaoVideoDocument{{Title: "영상", Author: "업로더", PlayTime: 30}}), "30초"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(tc.got, tc.want) {
				t.Fatalf("output = %s", tc.got)
			}
			if strings.Contains(tc.got, "<b>") {
				t.Fatal("HTML tag leaked")
			}
		})
	}
}

func TestFormatKakaoPlaces(t *testing.T) {
	got := formatKakaoPlaces([]kakaoPlaceDocument{{PlaceName: "카페", CategoryName: "음식점", RoadAddressName: "테스트로", PlaceURL: "https://place.example"}})
	for _, want := range []string{"카페", "음식점", "테스트로", "https://place.example"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
