package tools

import "fmt"

// normalizeKakaoSearch applies the limits documented by each Kakao collection.
// Keeping this pure makes API contract changes easy to drive with table tests.
func normalizeKakaoSearch(collection string, page, size int, sort string) (int, int, string, error) {
	if page <= 0 {
		page = 1
	}
	maxPage := 50
	if collection == "vclip" {
		maxPage = 15
	}
	if page > maxPage {
		return 0, 0, "", fmt.Errorf("page는 %d 이하이어야 합니다", maxPage)
	}
	if size <= 0 {
		size = 10
	}
	maxSize := 50
	if collection == "image" {
		maxSize = 80
	}
	if collection == "vclip" {
		maxSize = 30
	}
	if size > maxSize {
		return 0, 0, "", fmt.Errorf("size는 %d 이하이어야 합니다", maxSize)
	}
	if sort == "" {
		sort = "accuracy"
	}
	if collection == "book" {
		if sort != "accuracy" && sort != "latest" {
			return 0, 0, "", fmt.Errorf("책 sort는 accuracy 또는 latest이어야 합니다")
		}
	} else if sort != "accuracy" && sort != "recency" {
		return 0, 0, "", fmt.Errorf("sort는 accuracy 또는 recency이어야 합니다")
	}
	return page, size, sort, nil
}
