package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"wkqcosoft.com/public-api-mcp/config"
	"wkqcosoft.com/public-api-mcp/httpclient"
)

var envFile string
var debug bool

var RootCmd = &cobra.Command{
	Use:   "public-api-mcp",
	Short: "공공 API MCP 서버",
	Long:  "한국 공공 API 37개를 MCP 프로토콜로 노출하는 서버",
}

func init() {
	RootCmd.PersistentFlags().StringVar(&envFile, "env", "", ".env 파일 경로 (기본: .env)")
	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "DEBUG 레벨 로그 활성화")
	RootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		level := slog.LevelWarn
		if debug {
			level = slog.LevelDebug
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
		config.Load(envFile)
		httpclient.SetDebug(debug)
		return nil
	}
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
