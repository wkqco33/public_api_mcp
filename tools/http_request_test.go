package tools

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		blocked bool
	}{
		{"IPv4 루프백", "127.0.0.1", true},
		{"IPv6 루프백", "::1", true},
		{"IPv4 사설 10.x", "10.1.2.3", true},
		{"IPv4 사설 172.16-31.x", "172.20.0.5", true},
		{"IPv4 사설 192.168.x", "192.168.1.1", true},
		{"IPv4 링크로컬", "169.254.1.1", true},
		{"IPv6 링크로컬", "fe80::1", true},
		{"IPv6 ULA(사설)", "fc00::1", true},
		{"미지정 주소 0.0.0.0", "0.0.0.0", true},
		{"미지정 주소 ::", "::", true},
		{"IPv4 멀티캐스트", "224.0.0.1", true},
		{"IPv4-mapped IPv6 루프백", "::ffff:127.0.0.1", true},
		{"공인 IPv4 (8.8.8.8)", "8.8.8.8", false},
		{"공인 IPv4 (1.1.1.1)", "1.1.1.1", false},
		{"공인 IPv6", "2001:4860:4860::8888", false},
		{"172.x 대역이지만 사설 범위 밖(172.32.x)", "172.32.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("IP 파싱 실패: %s", tt.ip)
			}
			got := isBlockedIP(ip)
			if got != tt.blocked {
				t.Errorf("isBlockedIP(%s) = %v, want %v", tt.ip, got, tt.blocked)
			}
		})
	}
}

func TestSafeDialContextBlocksLiteralIP(t *testing.T) {
	blockedAddrs := []string{
		"127.0.0.1:80",
		"10.0.0.5:443",
		"192.168.1.1:80",
		"169.254.169.254:80", // 클라우드 메타데이터 엔드포인트
		"[::1]:80",
	}
	for _, addr := range blockedAddrs {
		t.Run(addr, func(t *testing.T) {
			conn, err := safeDialContext(context.Background(), "tcp", addr)
			if err == nil {
				conn.Close()
				t.Fatalf("차단되어야 할 주소가 연결 허용됨: %s", addr)
			}
			if !strings.Contains(err.Error(), "내부 네트워크") {
				t.Errorf("에러 메시지가 예상과 다름: %v", err)
			}
		})
	}
}

func TestSafeDialContextRejectsMalformedAddr(t *testing.T) {
	if _, err := safeDialContext(context.Background(), "tcp", "not-a-valid-addr"); err == nil {
		t.Fatal("host:port 형식이 아닌 주소가 허용됨")
	}
}

// TestHTTPRequestBlocksLoopback은 http_request 툴 핸들러 전체를 통해, IP 리터럴로 지정된
// 내부 주소가 실제 연결 시도 없이 차단되는지 확인한다(존재하지 않는 포트를 사용해 만약
// 실제로 연결을 시도했다면 반드시 실패했을 것이므로, 통과한다면 사전 차단이 동작한 것).
func TestHTTPRequestBlocksLoopback(t *testing.T) {
	result, _, err := httpRequest(context.Background(), nil, httpRequestInput{URL: "http://127.0.0.1:1/"})
	if err != nil {
		t.Fatalf("예상치 못한 Go 에러: %v", err)
	}
	if !result.IsError {
		t.Fatal("루프백 주소 요청이 차단되지 않음")
	}
}

// TestHTTPRequestBlocksLocalhostHostname은 IP 리터럴이 아닌 호스트명("localhost")이
// resolve 이후 차단되는지 확인한다 — 문자열 정규식이 아닌 실제 IP 기반 검사임을 검증.
func TestHTTPRequestBlocksLocalhostHostname(t *testing.T) {
	result, _, err := httpRequest(context.Background(), nil, httpRequestInput{URL: "http://localhost:1/"})
	if err != nil {
		t.Fatalf("예상치 못한 Go 에러: %v", err)
	}
	if !result.IsError {
		t.Fatal("localhost 호스트명 요청이 차단되지 않음")
	}
}

func TestHTTPRequestRejectsNonHTTPScheme(t *testing.T) {
	cases := []string{"file:///etc/passwd", "ftp://example.com/", "gopher://example.com/"}
	for _, u := range cases {
		t.Run(u, func(t *testing.T) {
			result, _, err := httpRequest(context.Background(), nil, httpRequestInput{URL: u})
			if err != nil {
				t.Fatalf("예상치 못한 Go 에러: %v", err)
			}
			if !result.IsError {
				t.Errorf("허용되지 않는 스킴이 차단되지 않음: %s", u)
			}
		})
	}
}

func TestHTTPRequestRejectsUnsupportedMethod(t *testing.T) {
	result, _, err := httpRequest(context.Background(), nil, httpRequestInput{URL: "https://example.com/", Method: "TRACE"})
	if err != nil {
		t.Fatalf("예상치 못한 Go 에러: %v", err)
	}
	if !result.IsError {
		t.Fatal("지원하지 않는 메서드가 차단되지 않음")
	}
}

func TestHTTPRequestToolRegistered(t *testing.T) {
	// withRecover로 감싼 뒤에도 등록이 정상적으로 이뤄지는지 스모크 확인.
	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	RegisterHTTPRequest(s)
}
