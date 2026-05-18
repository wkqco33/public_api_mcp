package tools

import (
	"regexp"
	"testing"
)

func TestFindLocationExists(t *testing.T) {
	loc := findLocation("서울")
	if loc == nil {
		t.Fatal("'서울' 검색 결과: nil — 기대: non-nil")
	}
	if loc.NX == "" || loc.NY == "" {
		t.Errorf("격자 좌표가 비어 있음: NX=%q, NY=%q", loc.NX, loc.NY)
	}
}

func TestFindLocationNotExists(t *testing.T) {
	loc := findLocation("없는지역xyzabc")
	if loc != nil {
		t.Fatalf("없는 지역에서 결과 반환됨: %+v", loc)
	}
}

func TestFindLocationStripNalsi(t *testing.T) {
	// "날씨" 포함 입력은 제거 후 검색
	loc := findLocation("서울날씨")
	if loc == nil {
		t.Fatal("'서울날씨' 검색 결과: nil — '날씨' 제거 후 검색되어야 함")
	}
}

func TestFindLocationStep3Priority(t *testing.T) {
	// 3단계(동/읍/면)로 검색 가능한 지역 (CSV에 "역삼1동"으로 등록됨)
	loc := findLocation("역삼1동")
	if loc == nil {
		t.Fatal("'역삼1동' 검색 결과: nil")
	}
}

func TestFindLocationGyeonggi(t *testing.T) {
	loc := findLocation("수원")
	if loc == nil {
		t.Fatal("'수원' 검색 결과: nil")
	}
}

func TestKmaBaseTimeNcstFormat(t *testing.T) {
	date, timeStr := kmaBaseTimeNcst()

	if len(date) != 8 {
		t.Errorf("날짜 길이 오류: got=%d, want=8 (yyyyMMdd)", len(date))
	}
	if len(timeStr) != 4 {
		t.Errorf("시간 길이 오류: got=%d, want=4 (HH00)", len(timeStr))
	}
	// 시간이 "MM00" 형식 — 끝 2자리가 "00"이어야 함
	if timeStr[2:] != "00" {
		t.Errorf("시간 분이 '00'이 아님: got=%q", timeStr[2:])
	}
}

func TestKmaBaseTimeVilageFormat(t *testing.T) {
	validTimes := regexp.MustCompile(`^(02|05|08|11|14|17|20|23)00$`)

	date, timeStr := kmaBaseTimeVilage()

	if len(date) != 8 {
		t.Errorf("날짜 길이 오류: got=%d, want=8", len(date))
	}
	if !validTimes.MatchString(timeStr) {
		t.Errorf("유효하지 않은 예보 시간: %q (02/05/08/11/14/17/20/23 중 하나여야 함)", timeStr)
	}
}
