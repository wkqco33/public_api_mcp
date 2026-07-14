package httpclient

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
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

var debugMu sync.Mutex

// SetDebug는 debug 모드를 활성화/비활성화한다.
// 활성화 시 모든 HTTP 요청의 URL, 상태코드, 소요시간이 DEBUG 레벨로 기록된다.
func SetDebug(on bool) {
	debugMu.Lock()
	defer debugMu.Unlock()
	if on {
		Default.Transport = debugRoundTripper{}
	} else {
		Default.Transport = nil
	}
}

// FetchBody는 req를 Default 클라이언트로 실행하고, 응답 바디를 limit 바이트까지만
// 읽어 반환한다. 상태 코드 해석과 바디 디코딩(JSON/XML 등)은 호출자의 책임이다 —
// 공공 API마다 성공 판정 방식과 응답 스키마가 달라 이 계층에서 단일화하기 어렵기
// 때문이다. tools 패키지 전반에 반복되던 "요청 → 크기 제한 → 바디 읽기" 보일러플레이트를
// 여기 한 곳으로 모은다.
func FetchBody(req *http.Request, limit int64) ([]byte, int, error) {
	res, err := Default.Do(req)
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
