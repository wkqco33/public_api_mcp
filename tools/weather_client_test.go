package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWeatherClientUsesInjectedHTTPClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getUltraSrtNcst" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("serviceKey") != "weather-test-key" {
			t.Errorf("service key not injected")
		}
		_, _ = w.Write([]byte(`{"response":{"header":{"resultCode":"00","resultMsg":"NORMAL_SERVICE"},"body":{"items":{"item":[{"category":"T1H","obsrValue":"21"}]}}}}`))
	}))
	defer ts.Close()
	client := NewWeatherClient(ts.Client(), ts.URL, "weather-test-key")
	body, err := kmaGetWithClient(context.Background(), client, "getUltraSrtNcst", map[string]string{"nx": "60", "ny": "127"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := body.Items.Item
	if len(list) != 1 || !strings.Contains(string(mustJSON(t, list[0])), "21") {
		t.Fatalf("unexpected decoded body: %#v", body)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
