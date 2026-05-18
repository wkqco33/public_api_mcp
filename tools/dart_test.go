package tools

import "testing"

// setupTestCorpCodes는 dartCorpCodes를 테스트용 데이터로 교체하고
// 테스트 종료 시 nil로 복원한다.
func setupTestCorpCodes(t *testing.T, codes map[string]string) {
	t.Helper()
	dartCorpCodesMu.Lock()
	dartCorpCodes = codes
	dartCorpCodesMu.Unlock()
	t.Cleanup(func() {
		dartCorpCodesMu.Lock()
		dartCorpCodes = nil
		dartCorpCodesMu.Unlock()
	})
}

func TestResolveCorpCodeEightDigits(t *testing.T) {
	// 8자리 숫자는 loadCorpCodes 호출 없이 직통 반환
	got, err := resolveCorpCode("00126380")
	if err != nil {
		t.Fatalf("예상치 못한 오류: %v", err)
	}
	if got != "00126380" {
		t.Errorf("got=%q, want=%q", got, "00126380")
	}
}

func TestResolveCorpCodeNonDigit8(t *testing.T) {
	// 8자리지만 숫자가 아닌 경우 → 기업명 검색으로 진행
	setupTestCorpCodes(t, map[string]string{
		"abc12345": "99999999",
	})

	got, err := resolveCorpCode("abc12345")
	if err != nil {
		t.Fatalf("오류: %v", err)
	}
	if got != "99999999" {
		t.Errorf("got=%q, want=%q", got, "99999999")
	}
}

func TestResolveCorpCodeExactMatch(t *testing.T) {
	setupTestCorpCodes(t, map[string]string{
		"삼성전자":  "00126380",
		"삼성SDI": "00126370",
	})

	got, err := resolveCorpCode("삼성전자")
	if err != nil {
		t.Fatalf("오류: %v", err)
	}
	if got != "00126380" {
		t.Errorf("got=%q, want=%q", got, "00126380")
	}
}

func TestResolveCorpCodePartialMatch(t *testing.T) {
	// 부분 일치 — 가장 짧은 이름 우선 반환
	setupTestCorpCodes(t, map[string]string{
		"삼성전자":          "00126380",
		"삼성전자서비스":       "00654321",
		"삼성전자판매":        "00789012",
	})

	got, err := resolveCorpCode("삼성전자")
	if err != nil {
		t.Fatalf("오류: %v", err)
	}
	// "삼성전자" 정확 일치가 존재하므로 해당 코드 반환
	if got != "00126380" {
		t.Errorf("got=%q, want=%q", got, "00126380")
	}
}

func TestResolveCorpCodeShortestFirst(t *testing.T) {
	// 정확 일치 없고 부분 일치만 있을 때 — 가장 짧은 이름 우선
	setupTestCorpCodes(t, map[string]string{
		"현대자동차":    "00164742",
		"현대자동차그룹":  "00999999",
	})

	got, err := resolveCorpCode("현대자")
	if err != nil {
		t.Fatalf("오류: %v", err)
	}
	// "현대자동차"(5자)가 "현대자동차그룹"(7자)보다 짧으므로 우선
	if got != "00164742" {
		t.Errorf("got=%q, want=%q", got, "00164742")
	}
}

func TestResolveCorpCodeNotFound(t *testing.T) {
	setupTestCorpCodes(t, map[string]string{
		"삼성전자": "00126380",
	})

	_, err := resolveCorpCode("없는기업xyz")
	if err == nil {
		t.Fatal("오류가 반환되어야 함 — got: nil")
	}
}
