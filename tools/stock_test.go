package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStockPriceWithInjectedClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/GetStockSecuritiesInfoService/getStockPriceInfo" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("serviceKey") != "stock-test-key" {
			t.Errorf("service key not injected")
		}
		_, _ = w.Write([]byte(`{"response":{"header":{"resultCode":"00"},"body":{"items":{"item":[{"basDt":"20240307","itmsNm":"삼성전자","srtnCd":"005930","clpr":"70000"}]}}}}`))
	}))
	defer ts.Close()
	client := NewStockClient(ts.Client(), ts.URL, "stock-test-key")
	result, _, err := getStockPriceWithClient(context.Background(), client, stockPriceInput{ItemName: "삼성전자"})
	if err != nil || result.IsError {
		t.Fatalf("unexpected result: %v", err)
	}
	if !strings.Contains(firstText(t, result), "삼성전자") {
		t.Fatalf("output = %s", firstText(t, result))
	}
}

func TestStockPriceMissingKey(t *testing.T) {
	client := NewStockClient(http.DefaultClient, "https://apis.data.go.kr/1160100/service", "")
	result, _, _ := getStockPriceWithClient(context.Background(), client, stockPriceInput{})
	if !result.IsError || !strings.Contains(firstText(t, result), "PUBLIC_DATA_PORTAL_API_KEY") {
		t.Fatalf("expected missing key error")
	}
}
