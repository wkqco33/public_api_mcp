package tools

import (
	"strings"
	"testing"
)

func TestFormatKakaoDocumentsByCollection(t *testing.T) {
	doc := map[string]any{
		"title": "<b>제목</b>", "contents": "A&amp;B", "url": "https://example.com",
		"blogname": "테스트 블로그", "authors": []any{"홍길동"}, "publisher": "출판사",
		"isbn": "1234", "image_url": "https://image.example", "display_sitename": "출처",
		"author": "업로더", "thumbnail": "https://thumb.example",
	}
	cases := []struct{ collection, want string }{
		{"blog", "테스트 블로그"}, {"book", "홍길동"}, {"image", "https://image.example"}, {"vclip", "업로더"},
	}
	for _, tc := range cases {
		t.Run(tc.collection, func(t *testing.T) {
			got := formatKakaoDocuments(tc.collection, []map[string]any{doc})
			if !strings.Contains(got, "제목") || !strings.Contains(got, "A&B") || !strings.Contains(got, tc.want) {
				t.Fatalf("formatted output = %s", got)
			}
			if strings.Contains(got, "<b>") {
				t.Fatal("HTML highlight tag should be removed")
			}
		})
	}
}

func TestFormatKakaoPlaces(t *testing.T) {
	got := formatKakaoPlaces([]map[string]any{{"place_name": "카페", "category_name": "음식점", "road_address_name": "테스트로", "place_url": "https://place.example"}})
	for _, want := range []string{"카페", "음식점", "테스트로", "https://place.example"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
