package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var allowedMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true,
	"DELETE": true, "PATCH": true, "HEAD": true, "OPTIONS": true,
}

var allowedSchemes = map[string]bool{"http": true, "https": true}

type httpRequestInput struct {
	URL     string `json:"url" jsonschema:"요청 URL"`
	Method  string `json:"method,omitempty" jsonschema:"HTTP 메서드: GET, POST, PUT, DELETE, PATCH (기본: GET)"`
	Headers string `json:"headers,omitempty" jsonschema:"JSON 형식 요청 헤더 (예: {\"Authorization\": \"Bearer token\"})"`
	Body    string `json:"body,omitempty" jsonschema:"요청 바디 (JSON 문자열 또는 일반 텍스트)"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"타임아웃 초 (기본: 30)"`
}

func RegisterHTTPRequest(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "http_request",
		Description: "임의의 HTTP 요청(GET/POST/PUT/DELETE 등)을 보내고 응답을 반환. API 테스트, webhook 호출에 사용. 내부 네트워크 주소는 차단됨.",
	}, withRecover("http_request", httpRequest))
}

// isBlockedIP는 사설망·루프백·링크로컬 등 내부용으로 예약된 주소인지 판별한다.
func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}

// safeDialContext는 호스트명을 실제로 접속하기 직전 시점에 IP로 resolve하여 검사한다.
// 검사(resolve)와 접속을 하나의 원자적 단계로 묶어 DNS 리바인딩(TOCTOU)을 막고,
// http.Transport의 DialContext로 등록되므로 리다이렉트로 이어지는 모든 hop에도
// 동일하게 적용되어 "허용된 외부 URL이 내부 주소로 302 리다이렉트"하는 우회도 차단한다.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}

	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("내부 네트워크 주소로의 요청은 허용되지 않습니다: %s", ip)
		}
		return dialer.DialContext(ctx, network, addr)
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("호스트를 확인할 수 없습니다: %s", host)
	}
	var chosen net.IP
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("내부 네트워크 주소로의 요청은 허용되지 않습니다: %s", host)
		}
		if chosen == nil {
			chosen = ip
		}
	}
	if chosen == nil {
		return nil, fmt.Errorf("호스트를 확인할 수 없습니다: %s", host)
	}
	return dialer.DialContext(ctx, network, net.JoinHostPort(chosen.String(), port))
}

func httpRequest(ctx context.Context, _ *mcp.CallToolRequest, in httpRequestInput) (*mcp.CallToolResult, any, error) {
	method := strings.ToUpper(in.Method)
	if method == "" {
		method = "GET"
	}
	if !allowedMethods[method] {
		return toolError("지원하지 않는 HTTP 메서드: %s", method), nil, nil
	}

	parsed, err := url.Parse(in.URL)
	if err != nil || !allowedSchemes[strings.ToLower(parsed.Scheme)] {
		return toolError("http 또는 https URL만 허용됩니다"), nil, nil
	}

	var bodyReader io.Reader
	if in.Body != "" {
		bodyReader = strings.NewReader(in.Body)
	}

	timeout := in.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	client := &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: &http.Transport{DialContext: safeDialContext},
	}

	req, err := http.NewRequestWithContext(ctx, method, in.URL, bodyReader)
	if err != nil {
		return toolError("요청 생성 실패: %v", err), nil, nil
	}

	if in.Headers != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(in.Headers), &headers); err != nil {
			return toolError("headers가 올바른 JSON 형식이 아닙니다: %v", err), nil, nil
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	// Body가 JSON이면 Content-Type 자동 설정
	if in.Body != "" && req.Header.Get("Content-Type") == "" {
		var js any
		if json.Unmarshal([]byte(in.Body), &js) == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	res, err := client.Do(req)
	if err != nil {
		return toolError("요청 오류: %v", err), nil, nil
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(res.Body, 4000))
	if err != nil {
		return toolError("응답 읽기 실패: %v", err), nil, nil
	}
	contentType := res.Header.Get("Content-Type")
	preview := string(respBody)
	if len(preview) > 4000 {
		preview = preview[:4000] + "..."
	}

	result := fmt.Sprintf("HTTP %d %s\nContent-Type: %s\n\n%s", res.StatusCode, res.Status, contentType, preview)
	return toolText(result), nil, nil
}
