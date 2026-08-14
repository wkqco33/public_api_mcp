package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchBusStopWithInjectedClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("serviceKey") != "bus-key" {
			t.Errorf("service key not injected")
		}
		_, _ = w.Write([]byte(`{"response":{"body":{"items":{"item":[{"nodeid":"123","nodenm":"강남역"}]}}}}`))
	}))
	defer ts.Close()
	client := NewBusClient(ts.Client(), ts.URL, "bus-key")
	result, _, err := searchBusStopWithClient(context.Background(), client, busStopSearchInput{StopName: "강남역"})
	if err != nil || result.IsError || !strings.Contains(firstText(t, result), "강남역") {
		t.Fatalf("unexpected result: %v", err)
	}
}

func TestGetSubwayArrivalWithInjectedClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/subway-key/json/") {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"realtimeArrivalList":[{"subwayId":"2","trainLineNm":"내선","bstatnNm":"삼성","arvlMsg2":"강남역 진입"}]}`))
	}))
	defer ts.Close()
	client := NewSubwayClient(ts.Client(), ts.URL, "subway-key")
	result, _, err := getSubwayArrivalWithClient(context.Background(), client, subwayArrivalInput{StationName: "강남"})
	if err != nil || result.IsError || !strings.Contains(firstText(t, result), "강남") {
		t.Fatalf("unexpected result: %v", err)
	}
}
