package httpclient

import (
	"log/slog"
	"net/http"
	"time"
)

const UserAgent = "Mozilla/5.0 (compatible; public-api-mcp/1.0)"

var Default = &http.Client{
	Timeout: 30 * time.Second,
}

type debugRoundTripper struct{}

func (debugRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		slog.Debug("HTTP 실패", "method", req.Method, "url", req.URL.String(), "err", err)
		return nil, err
	}
	slog.Debug("HTTP", "method", req.Method, "url", req.URL.String(),
		"status", resp.StatusCode, "ms", time.Since(start).Milliseconds())
	return resp, nil
}

// SetDebug는 debug 모드를 활성화/비활성화한다.
// 활성화 시 모든 HTTP 요청의 URL, 상태코드, 소요시간이 DEBUG 레벨로 기록된다.
func SetDebug(on bool) {
	if on {
		Default.Transport = debugRoundTripper{}
	} else {
		Default.Transport = nil
	}
}
