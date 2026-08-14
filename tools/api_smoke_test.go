package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"wkqcosoft.com/public-api-mcp/config"
	"wkqcosoft.com/public-api-mcp/httpclient"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// useTestServer는 httpclient.Default가 실제 외부 호스트 대신 ts로 요청을 보내도록
// RoundTripper를 임시로 교체한다. 각 tools 파일이 호출하는 API 엔드포인트는
// 하드코딩된 외부 URL이므로, 요청 시점에 호스트만 테스트 서버로 바꿔치기해
// 프로덕션 코드를 건드리지 않고도 성공/실패 응답 처리 로직을 검증한다.
func useTestServer(t *testing.T, ts *httptest.Server) {
	t.Helper()
	testURL, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("테스트 서버 URL 파싱 실패: %v", err)
	}
	orig := httpclient.Default.Transport
	httpclient.Default.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.URL.Scheme = testURL.Scheme
		req.URL.Host = testURL.Host
		return http.DefaultTransport.RoundTrip(req)
	})
	t.Cleanup(func() { httpclient.Default.Transport = orig })
}

func firstText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil || len(result.Content) == 0 {
		t.Fatal("Content가 비어 있음")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Content[0]가 TextContent가 아님")
	}
	return tc.Text
}

func TestGetExchangeRatesSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"cur_unit":"USD","cur_nm":"미국 달러","deal_bas_r":"1,300.00","ttb":"1,287.00","tts":"1,313.00"}]`))
	}))
	defer ts.Close()
	useTestServer(t, ts)

	orig := config.ExchangeAPIKey
	config.ExchangeAPIKey = "test-key"
	t.Cleanup(func() { config.ExchangeAPIKey = orig })

	result, _, err := getExchangeRates(context.Background(), nil, exchangeInput{Date: "19990101"})
	if err != nil {
		t.Fatalf("예상치 못한 Go 에러: %v", err)
	}
	if result.IsError {
		t.Fatalf("에러 결과: %s", firstText(t, result))
	}
	text := firstText(t, result)
	if !strings.Contains(text, "USD") || !strings.Contains(text, "1,300.00") {
		t.Errorf("환율 정보가 응답에 포함되지 않음: %s", text)
	}
}

func TestGetExchangeRatesNoData(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer ts.Close()
	useTestServer(t, ts)

	orig := config.ExchangeAPIKey
	config.ExchangeAPIKey = "test-key"
	t.Cleanup(func() { config.ExchangeAPIKey = orig })

	result, _, err := getExchangeRates(context.Background(), nil, exchangeInput{Date: "19990102"})
	if err != nil {
		t.Fatalf("예상치 못한 Go 에러: %v", err)
	}
	if !result.IsError {
		t.Fatalf("데이터 없음(7일 탐색 후에도 빈 배열)인 경우 에러여야 함: %s", firstText(t, result))
	}
}

func TestGetRealEstateDataSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<response><body><items><item><aptNm>테스트아파트</aptNm><dealAmount>50000</dealAmount></item></items></body></response>`))
	}))
	defer ts.Close()
	useTestServer(t, ts)

	orig := config.PublicDataPortalAPIKey
	config.PublicDataPortalAPIKey = "test-key"
	t.Cleanup(func() { config.PublicDataPortalAPIKey = orig })

	result, _, err := getRealEstateData(context.Background(), nil, realEstateInput{
		APIType: "apt_trade",
		LawdCd:  "11110",
		DealYmd: "202405",
	})
	if err != nil {
		t.Fatalf("예상치 못한 Go 에러: %v", err)
	}
	if result.IsError {
		t.Fatalf("에러 결과: %s", firstText(t, result))
	}
	text := firstText(t, result)
	if !strings.Contains(text, "테스트아파트") {
		t.Errorf("실거래가 정보가 응답에 포함되지 않음: %s", text)
	}
}

// TestSearchPixabayImagesMalformedHit는 hit["id"]가 기대한 float64가 아니거나
// 아예 없는 비정상 응답을 받아도 패닉 없이 결과를 반환하는지 확인한다
// (과거 무조건 타입 단언 hit["id"].(float64)이 이런 입력에서 패닉을 일으켰다).
func TestSearchPixabayImagesMalformedHit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"hits":[{"previewURL":"http://example.com/p.jpg","tags":"cat","user":"tester"},{"id":"not-a-number","previewURL":"http://example.com/q.jpg"}]}`))
	}))
	defer ts.Close()
	useTestServer(t, ts)

	orig := config.PixabayAPIKey
	config.PixabayAPIKey = "test-key"
	t.Cleanup(func() { config.PixabayAPIKey = orig })

	result, _, err := searchPixabayImages(context.Background(), nil, pixabayInput{Query: "cat"})
	if err != nil {
		t.Fatalf("예상치 못한 Go 에러: %v", err)
	}
	if result.IsError {
		t.Fatalf("비정상 id 필드만으로 에러가 되어서는 안 됨: %s", firstText(t, result))
	}
	if !strings.Contains(firstText(t, result), "cat") {
		t.Errorf("정상 필드는 여전히 응답에 포함되어야 함: %s", firstText(t, result))
	}
}

func TestGetRealEstateDataNoItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<response><body><items></items></body></response>`))
	}))
	defer ts.Close()
	useTestServer(t, ts)

	orig := config.PublicDataPortalAPIKey
	config.PublicDataPortalAPIKey = "test-key"
	t.Cleanup(func() { config.PublicDataPortalAPIKey = orig })

	result, _, err := getRealEstateData(context.Background(), nil, realEstateInput{
		APIType: "apt_trade",
		LawdCd:  "11110",
		DealYmd: "202405",
	})
	if err != nil {
		t.Fatalf("예상치 못한 Go 에러: %v", err)
	}
	if result.IsError {
		t.Fatalf("데이터 없음은 에러가 아닌 안내 메시지여야 함: %s", firstText(t, result))
	}
	if !strings.Contains(firstText(t, result), "데이터가 없습니다") {
		t.Errorf("빈 결과 안내 메시지가 없음: %s", firstText(t, result))
	}
}
