package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wkqcosoft.com/public-api-mcp/config"
)

func TestKakaoSearchSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/search/blog" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "KakaoAK test-key" {
			t.Errorf("authorization header missing")
		}
		if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("size") != "5" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"total_count":1,"pageable_count":1,"is_end":true},"documents":[{"title":"<b>테스트</b>","contents":"A&amp;B","url":"https://example.com","blogname":"블로그"}]}`))
	}))
	defer ts.Close()
	useTestServer(t, ts)
	orig := config.KakaoRESTAPIKey
	config.KakaoRESTAPIKey = "test-key"
	t.Cleanup(func() { config.KakaoRESTAPIKey = orig })

	result, _, err := kakaoSearch(context.Background(), "blog", kakaoSearchInput{Query: "go", Page: 2, Size: 5})
	if err != nil || result.IsError {
		t.Fatalf("unexpected error: %v %s", err, firstText(t, result))
	}
	text := firstText(t, result)
	if !strings.Contains(text, "테스트") || !strings.Contains(text, "A&B") {
		t.Errorf("formatted result = %s", text)
	}
}

func TestKakaoSearchValidation(t *testing.T) {
	if _, _, err := kakaoSearch(context.Background(), "blog", kakaoSearchInput{Query: "", Size: 1}); err != nil {
		t.Fatalf("validation should be tool result, not Go error: %v", err)
	}
	result, _, _ := kakaoSearch(context.Background(), "blog", kakaoSearchInput{Query: "x", Size: 51})
	if !result.IsError || !strings.Contains(firstText(t, result), "size") {
		t.Fatalf("expected size validation error")
	}
}

func TestKakaoPlacesValidation(t *testing.T) {
	result, _, _ := kakaoSearchPlaces(context.Background(), nil, kakaoPlaceInput{Query: "카페", Radius: 1000})
	if !result.IsError || !strings.Contains(firstText(t, result), "longitude") {
		t.Fatalf("expected coordinate validation error")
	}
}

func TestKakaoSearchCollectionsUseExpectedEndpoints(t *testing.T) {
	collections := []struct{ collection, path string }{
		{"web", "/v2/search/web"}, {"cafe", "/v2/search/cafe"},
		{"book", "/v3/search/book"}, {"image", "/v2/search/image"},
		{"vclip", "/v2/search/vclip"},
	}
	for _, tc := range collections {
		t.Run(tc.collection, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.path {
					t.Errorf("path = %s, want %s", r.URL.Path, tc.path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"meta":{},"documents":[{"title":"result","url":"https://example.com"}]}`))
			}))
			defer ts.Close()
			useTestServer(t, ts)
			orig := config.KakaoRESTAPIKey
			config.KakaoRESTAPIKey = "test-key"
			t.Cleanup(func() { config.KakaoRESTAPIKey = orig })
			result, _, err := kakaoSearch(context.Background(), tc.collection, kakaoSearchInput{Query: "test"})
			if err != nil || result.IsError {
				t.Fatalf("unexpected result: %v", err)
			}
		})
	}
}

func TestKakaoSearchErrors(t *testing.T) {
	orig := config.KakaoRESTAPIKey
	config.KakaoRESTAPIKey = ""
	result, _, _ := kakaoSearch(context.Background(), "web", kakaoSearchInput{Query: "test"})
	config.KakaoRESTAPIKey = orig
	if !result.IsError || !strings.Contains(firstText(t, result), "KAKAO_REST_API_KEY") {
		t.Fatal("missing API key should be reported")
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"msg":"quota exceeded"}`))
	}))
	defer ts.Close()
	useTestServer(t, ts)
	config.KakaoRESTAPIKey = "test-key"
	t.Cleanup(func() { config.KakaoRESTAPIKey = orig })
	result, _, _ = kakaoSearch(context.Background(), "web", kakaoSearchInput{Query: "test"})
	if !result.IsError || !strings.Contains(firstText(t, result), "429") {
		t.Fatal("429 should be returned as a tool error")
	}
}

func TestKakaoClientCanUseInjectedHTTPClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/search/web" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "KakaoAK injected-key" {
			t.Errorf("missing injected authorization")
		}
		_, _ = w.Write([]byte(`{"meta":{},"documents":[{"title":"injected"}]}`))
	}))
	defer ts.Close()
	client := newKakaoClient(ts.Client(), ts.URL, "injected-key")
	result, _, err := kakaoSearchRequestWithClient(context.Background(), client, "web", "test", 1, 1, "accuracy")
	if err != nil || result.IsError || !strings.Contains(firstText(t, result), "injected") {
		t.Fatalf("unexpected injected client result: %v", err)
	}
}

func TestKakaoPlacesSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("x") != "127.1" || r.URL.Query().Get("radius") != "1000" {
			t.Errorf("unexpected place query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{},"documents":[{"place_name":"테스트 카페","category_name":"카페","x":"127.1","y":"37.5","place_url":"https://place.example"}]}`))
	}))
	defer ts.Close()
	useTestServer(t, ts)
	orig := config.KakaoRESTAPIKey
	config.KakaoRESTAPIKey = "test-key"
	t.Cleanup(func() { config.KakaoRESTAPIKey = orig })
	result, _, err := kakaoSearchPlaces(context.Background(), nil, kakaoPlaceInput{Query: "카페", Longitude: 127.1, Latitude: 37.5, Radius: 1000})
	if err != nil || result.IsError || !strings.Contains(firstText(t, result), "테스트 카페") {
		t.Fatalf("unexpected place result: %v", err)
	}
}
