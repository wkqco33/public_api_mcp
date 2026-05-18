package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var blockedHosts = regexp.MustCompile(
	`(?i)^(localhost|127\.\d+\.\d+\.\d+|0\.0\.0\.0` +
		`|10\.\d+\.\d+\.\d+` +
		`|192\.168\.\d+\.\d+` +
		`|172\.(1[6-9]|2\d|3[01])\.\d+\.\d+` +
		`|169\.254\.\d+\.\d+` +
		`|::1|fc00:|fe80:)`,
)

var allowedMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true,
	"DELETE": true, "PATCH": true, "HEAD": true, "OPTIONS": true,
}

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
	}, httpRequest)
}

func httpRequest(_ context.Context, _ *mcp.CallToolRequest, in httpRequestInput) (*mcp.CallToolResult, any, error) {
	method := strings.ToUpper(in.Method)
	if method == "" {
		method = "GET"
	}
	if !allowedMethods[method] {
		return toolError("지원하지 않는 HTTP 메서드: %s", method), nil, nil
	}

	parsed, err := url.Parse(in.URL)
	if err != nil || blockedHosts.MatchString(parsed.Hostname()) {
		return toolError("내부 네트워크 주소로의 요청은 허용되지 않습니다"), nil, nil
	}

	var bodyReader io.Reader
	if in.Body != "" {
		bodyReader = strings.NewReader(in.Body)
	}

	timeout := in.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}

	req, err := http.NewRequest(method, in.URL, bodyReader)
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

	respBody, _ := io.ReadAll(io.LimitReader(res.Body, 4000))
	contentType := res.Header.Get("Content-Type")
	preview := string(respBody)
	if len(preview) > 4000 {
		preview = preview[:4000] + "..."
	}

	result := fmt.Sprintf("HTTP %d %s\nContent-Type: %s\n\n%s", res.StatusCode, res.Status, contentType, preview)
	return toolText(result), nil, nil
}
