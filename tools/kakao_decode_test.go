package tools

import (
	"strings"
	"testing"
)

func TestFormatKakaoSearchResponseDecodesTypedDocuments(t *testing.T) {
	body := []byte(`{"meta":{"total_count":1},"documents":[{"title":"책","authors":["저자1","저자2"],"publisher":"출판사","isbn":"123"}]}`)
	got, empty, err := formatKakaoSearchResponse("book", body)
	if err != nil || empty {
		t.Fatalf("decode result: empty=%v err=%v", empty, err)
	}
	if !strings.Contains(got, "저자1, 저자2") || !strings.Contains(got, "출판사") {
		t.Fatalf("formatted typed result = %s", got)
	}
}

func TestFormatKakaoSearchResponseRejectsMalformedAndUnknown(t *testing.T) {
	if _, _, err := formatKakaoSearchResponse("web", []byte("not-json")); err == nil {
		t.Fatal("malformed JSON should fail")
	}
	if _, _, err := formatKakaoSearchResponse("unknown", []byte(`{"documents":[]}`)); err == nil {
		t.Fatal("unknown collection should fail")
	}
}
