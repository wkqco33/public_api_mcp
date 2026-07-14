package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAuth(t *testing.T) {
	const token = "test-secret-token"

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := requireAuth(token, next)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"헤더 없음", "", http.StatusUnauthorized},
		{"Bearer 접두사 없음", token, http.StatusUnauthorized},
		{"틀린 토큰", "Bearer wrong-token", http.StatusUnauthorized},
		{"빈 Bearer 토큰", "Bearer ", http.StatusUnauthorized},
		{"정확한 토큰", "Bearer " + token, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusUnauthorized {
				if got := rec.Header().Get("WWW-Authenticate"); got != "Bearer" {
					t.Errorf("WWW-Authenticate = %q, want %q", got, "Bearer")
				}
			}
		})
	}
}

func TestRequireAuthDoesNotPanicOnEmptyToken(t *testing.T) {
	// requireAuth 자체는 빈 토큰으로도 패닉 없이 동작해야 한다(서버 시작을 막는 로직은
	// serveHTTPCmd.RunE 쪽 책임이라 별도로 검증됨). 다만 빈 토큰이면 어떤 헤더값도
	// 일치할 수 없어야 한다(길이 0 vs 0 비교로 우회되지 않는지 확인).
	handler := requireAuth("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("빈 토큰 설정 시 빈 Authorization으로 통과되면 안 됨: status = %d", rec.Code)
	}
}
