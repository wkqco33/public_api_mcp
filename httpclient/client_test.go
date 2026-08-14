package httpclient

import (
	"net/url"
	"strings"
	"testing"
)

func TestRedactedURLHidesSecrets(t *testing.T) {
	u, err := url.Parse("https://example.com/api?serviceKey=secret&query=test&token=abc")
	if err != nil {
		t.Fatal(err)
	}
	got := redactedURL(u)
	if strings.Contains(got, "secret") || strings.Contains(got, "abc") {
		t.Fatalf("secret leaked: %s", got)
	}
	if !strings.Contains(got, "query=test") || !strings.Contains(got, "%5BREDACTED%5D") {
		t.Fatalf("unexpected redacted URL: %s", got)
	}
}
