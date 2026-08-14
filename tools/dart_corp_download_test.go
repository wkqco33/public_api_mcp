package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPDARTCorpCodeDownloader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/corpCode.xml" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("crtfc_key") != "dart-dl-key" {
			t.Errorf("certificate key missing")
		}
		_, _ = w.Write([]byte("zip-bytes"))
	}))
	defer ts.Close()
	client := NewDARTClient(ts.Client(), ts.URL, "dart-dl-key")
	data, err := (&httpDARTCorpCodeDownloader{client: client}).Download(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "zip-bytes" {
		t.Fatalf("unexpected body: %q", data)
	}
}

func TestHTTPDARTCorpCodeDownloaderRequiresKey(t *testing.T) {
	client := NewDARTClient(http.DefaultClient, "https://opendart.fss.or.kr", "")
	if _, err := (&httpDARTCorpCodeDownloader{client: client}).Download(context.Background()); err == nil {
		t.Fatal("missing key should fail")
	}
}
