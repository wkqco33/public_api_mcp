package cmd

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"wkqcosoft.com/public-api-mcp/tools"
)

var addr string

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
		handler := mcp.NewSSEHandler(func(_ *http.Request) *mcp.Server { return s }, nil)
		slog.Info("HTTP MCP 서버 시작", "addr", addr)
		return http.ListenAndServe(addr, handler)
	},
}

func init() {
	serveHTTPCmd.Flags().StringVar(&addr, "addr", ":8080", "HTTP 리슨 주소")
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
