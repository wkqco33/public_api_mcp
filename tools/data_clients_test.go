package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFuelAvgPriceWithInjectedClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("code") != "fuel-key" {
			t.Errorf("fuel key not injected")
		}
		_, _ = w.Write([]byte(`{"RESULT":{"OIL":[{"PRODNM":"휘발유","PRICE":"1800.5"}]}}`))
	}))
	defer ts.Close()
	client := NewOpinetClient(ts.Client(), ts.URL, "fuel-key")
	result, _, err := getFuelAvgPriceWithClient(context.Background(), client, fuelAvgInput{})
	if err != nil || result.IsError || !strings.Contains(firstText(t, result), "휘발유") {
		t.Fatalf("unexpected result: %v", err)
	}
}

func TestGetRecommendedBooksWithInjectedClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("Key") != "book-key" {
			t.Errorf("book key not injected")
		}
		_, _ = w.Write([]byte(`<books><totalCount>1</totalCount><item><recomtitle>테스트 도서</recomtitle><recomauthor>저자</recomauthor><recompublisher>출판사</recompublisher></item></books>`))
	}))
	defer ts.Close()
	client := NewNLCClient(ts.Client(), ts.URL, "book-key")
	result, _, err := getRecommendedBooksWithClient(context.Background(), client, bookInput{})
	if err != nil || result.IsError || !strings.Contains(firstText(t, result), "테스트 도서") {
		t.Fatalf("unexpected result: %v", err)
	}
}

func TestGetRealEstateDataWithInjectedClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("serviceKey") != "estate-key" {
			t.Errorf("estate key not injected")
		}
		_, _ = w.Write([]byte(`<response><body><items><item><aptNm>테스트아파트</aptNm></item></items></body></response>`))
	}))
	defer ts.Close()
	client := NewRealEstateClient(ts.Client(), ts.URL, "estate-key")
	result, _, err := getRealEstateDataWithClient(context.Background(), client, realEstateInput{APIType: "apt_trade", LawdCd: "11110", DealYmd: "202405"})
	if err != nil || result.IsError || !strings.Contains(firstText(t, result), "테스트아파트") {
		t.Fatalf("unexpected result: %v", err)
	}
}

func TestSearchPixabayImagesWithInjectedClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "pixabay-key" {
			t.Errorf("pixabay key not injected")
		}
		_, _ = w.Write([]byte(`{"hits":[{"id":1,"previewURL":"https://example.com/p.jpg","tags":"cat","user":"tester"}]}`))
	}))
	defer ts.Close()
	client := NewPixabayClient(ts.Client(), ts.URL, "pixabay-key")
	result, _, err := searchPixabayImagesWithClient(context.Background(), client, pixabayInput{Query: "cat"})
	if err != nil || result.IsError || !strings.Contains(firstText(t, result), "cat") {
		t.Fatalf("unexpected result: %v", err)
	}
}
