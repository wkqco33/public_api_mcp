package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fixedExchangeClock struct{ t time.Time }

func (c fixedExchangeClock) Now() time.Time { return c.t }

func TestExchangeDependenciesAreInjectable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("authkey") != "injected-key" {
			t.Errorf("auth key was not injected")
		}
		_, _ = w.Write([]byte(`[{"cur_unit":"USD","cur_nm":"미국 달러","deal_bas_r":"1300","ttb":"1290","tts":"1310"}]`))
	}))
	defer ts.Close()
	client := NewExchangeClient(ts.Client(), ts.URL, "injected-key")
	result, _, err := getExchangeRatesWithDependencies(context.Background(), client, fixedExchangeClock{t: time.Date(2024, 3, 7, 12, 0, 0, 0, time.UTC)}, exchangeInput{})
	if err != nil || result.IsError {
		t.Fatalf("unexpected result: %v", err)
	}
	if !strings.Contains(firstText(t, result), "USD") || !strings.Contains(firstText(t, result), "1300") {
		t.Fatalf("unexpected exchange output: %s", firstText(t, result))
	}
}
