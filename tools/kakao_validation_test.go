package tools

import "testing"

func TestNormalizeKakaoSearchDefaults(t *testing.T) {
	page, size, sort, err := normalizeKakaoSearch("blog", 0, 0, "")
	if err != nil || page != 1 || size != 10 || sort != "accuracy" {
		t.Fatalf("got (%d, %d, %q, %v)", page, size, sort, err)
	}
}

func TestNormalizeKakaoSearchCollectionLimits(t *testing.T) {
	if _, _, _, err := normalizeKakaoSearch("vclip", 16, 10, ""); err == nil {
		t.Fatal("vclip page 16 should fail")
	}
	if _, _, _, err := normalizeKakaoSearch("image", 1, 81, ""); err == nil {
		t.Fatal("image size 81 should fail")
	}
	if _, _, _, err := normalizeKakaoSearch("book", 1, 10, "recency"); err == nil {
		t.Fatal("book recency should fail")
	}
	if _, _, _, err := normalizeKakaoSearch("web", 1, 10, "latest"); err == nil {
		t.Fatal("web latest should fail")
	}
}
