package cmd

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"wkqcosoft.com/public-api-mcp/config"
	"wkqcosoft.com/public-api-mcp/tools"
)

var addr string
var insecure bool

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "MCP 서버 시작",
}

var serveStdioCmd = &cobra.Command{
	Use:   "stdio",
	Short: "stdio 전송으로 서버 시작 (Claude Desktop 용)",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := buildServer()
		if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			slog.Error("서버 종료", "err", err)
		}
		return nil
	},
}

var serveHTTPCmd = &cobra.Command{
	Use:   "http",
	Short: "HTTP/SSE 전송으로 서버 시작",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := buildServer()
		handler := mcp.NewSSEHandler(func(_ *http.Request) *mcp.Server { return s }, &mcp.SSEOptions{
			// 리버스 프록시 뒤에서 운영할 때 Host 헤더가 외부 도메인으로 오므로 비활성화
			DisableLocalhostProtection: true,
		})

		var finalHandler http.Handler = handler
		switch {
		case config.MCPAuthToken != "":
			finalHandler = requireAuth(config.MCPAuthToken, handler)
		case insecure:
			slog.Warn("MCP_AUTH_TOKEN이 설정되지 않아 인증 없이 서버를 시작합니다 (--insecure)")
		default:
			return fmt.Errorf("MCP_AUTH_TOKEN이 설정되지 않았습니다 — 'public-api-mcp config generate-token'으로 토큰을 생성해 설정하거나, 로컬 테스트 목적이면 --insecure 플래그를 사용하세요")
		}

		slog.Info("HTTP MCP 서버 시작", "addr", addr)
		return http.ListenAndServe(addr, finalHandler)
	},
}

// requireAuth는 Authorization: Bearer <token> 헤더를 검사하는 미들웨어다.
// 상수 시간 비교로 타이밍 공격을 방지한다.
func requireAuth(token string, next http.Handler) http.Handler {
	const prefix = "Bearer "
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if token == "" || !strings.HasPrefix(authHeader, prefix) ||
			subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(authHeader, prefix)), []byte(token)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func init() {
	serveHTTPCmd.Flags().StringVar(&addr, "addr", ":8080", "HTTP 리슨 주소")
	serveHTTPCmd.Flags().BoolVar(&insecure, "insecure", false, "MCP_AUTH_TOKEN 없이 인증 없는 상태로 시작 (로컬 테스트 전용, 공개 노출 금지)")
	serveCmd.AddCommand(serveStdioCmd, serveHTTPCmd)
	RootCmd.AddCommand(serveCmd)
}

func buildServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "public-api-mcp", Version: "0.1.0"}, nil)
	tools.RegisterNaver(s)
	tools.RegisterGitHub(s)
	tools.RegisterExchange(s)
	tools.RegisterFuel(s)
	tools.RegisterStock(s)
	tools.RegisterRealEstate(s)
	tools.RegisterTransport(s)
	tools.RegisterWeather(s)
	tools.RegisterDART(s)
	tools.RegisterWeb(s)
	tools.RegisterBook(s)
	tools.RegisterPixabay(s)
	tools.RegisterHTTPRequest(s)
	return s
}
