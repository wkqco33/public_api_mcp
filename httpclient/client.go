package httpclient

import (
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const UserAgent = "Mozilla/5.0 (compatible; public-api-mcp/1.0)"

var customTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	TLSClientConfig: &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		},
	},
}

var Default = &http.Client{
	Timeout:   30 * time.Second,
	Transport: customTransport,
}

type debugRoundTripper struct{}

func (debugRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := customTransport.RoundTrip(req)
	if err != nil {
		slog.Debug("HTTP 실패", "method", req.Method, "url", redactedURL(req.URL), "err", err)
		return nil, err
	}
	slog.Debug("HTTP", "method", req.Method, "url", redactedURL(req.URL),
		"status", resp.StatusCode, "ms", time.Since(start).Milliseconds())
	return resp, nil
}

var debugMu sync.Mutex

func redactedURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	copy := *u
	q := copy.Query()
	for _, key := range []string{"key", "apiKey", "authKey", "serviceKey", "crtfc_key", "token", "access_token"} {
		if q.Has(key) {
			q.Set(key, "[REDACTED]")
		}
	}
	copy.RawQuery = q.Encode()
	return copy.String()
}

// SetDebug는 debug 모드를 활성화/비활성화한다.
// 활성화 시 모든 HTTP 요청의 URL, 상태코드, 소요시간이 DEBUG 레벨로 기록된다.
func SetDebug(on bool) {
	debugMu.Lock()
	defer debugMu.Unlock()
	if on {
		Default.Transport = debugRoundTripper{}
	} else {
		Default.Transport = customTransport
	}
}

// FetchBody는 req를 Default 클라이언트로 실행하고, 응답 바디를 limit 바이트까지만
// 읽어 반환한다. 상태 코드 해석과 바디 디코딩(JSON/XML 등)은 호출자의 책임이다 —
// 공공 API마다 성공 판정 방식과 응답 스키마가 달라 이 계층에서 단일화하기 어렵기
// 때문이다. tools 패키지 전반에 반복되던 "요청 → 크기 제한 → 바디 읽기" 보일러플레이트를
// 여기 한 곳으로 모은다.
func FetchBody(req *http.Request, limit int64) ([]byte, int, error) {
	return FetchBodyWithClient(Default, req, limit)
}

// FetchBodyWithClient voert req uit met een geïnjecteerde HTTP client.
// Hierdoor kunnen API-adapters zonder globale transportvervanging worden getest.
func FetchBodyWithClient(client interface {
	Do(*http.Request) (*http.Response, error)
}, req *http.Request, limit int64) ([]byte, int, error) {
	res, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("요청 실패: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, limit))
	if err != nil {
		return nil, res.StatusCode, fmt.Errorf("응답 읽기 실패: %w", err)
	}
	return body, res.StatusCode, nil
}
