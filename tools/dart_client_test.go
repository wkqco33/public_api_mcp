package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDARTClientUsesInjectedHTTPClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/company.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("crtfc_key") != "dart-test-key" {
			t.Errorf("certificate key was not injected")
		}
		_, _ = w.Write([]byte(`{"status":"000","corp_name":"테스트기업"}`))
	}))
	defer ts.Close()
	client := NewDARTClient(ts.Client(), ts.URL, "dart-test-key")
	data, err := dartGetWithClient(context.Background(), client, "/api/company.json", map[string]string{"corp_code": "00123456"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["corp_name"] != "테스트기업" {
		t.Fatalf("unexpected response: %#v", data)
	}
}

func TestDARTClientErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ERR","message":"bad key"}`))
	}))
	defer ts.Close()
	client := NewDARTClient(ts.Client(), ts.URL, "dart-test-key")
	if _, err := dartGetWithClient(context.Background(), client, "/api/company.json", nil); err == nil {
		t.Fatal("DART error response should return an error")
	}
}
