package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/spf13/cobra"
	"wkqcosoft.com/public-api-mcp/config"
)

type keyEntry struct {
	label  string
	getter func() string
}

var apiKeys = []keyEntry{
	{"KAKAO_REST_API_KEY", func() string { return config.KakaoRESTAPIKey }},
	{"EXCHANGE_API_KEY", func() string { return config.ExchangeAPIKey }},
	{"PUBLIC_DATA_PORTAL_API_KEY", func() string { return config.PublicDataPortalAPIKey }},
	{"SEOUL_OPEN_API_KEY", func() string { return config.SeoulOpenAPIKey }},
	{"KMA_WEATHER_API_KEY", func() string { return config.KMAWeatherAPIKey }},
	{"OPENDART_API_KEY", func() string { return config.OpenDARTAPIKey }},
	{"OPINET_API_KEY", func() string { return config.OpinetAPIKey }},
	{"NATIONAL_LIBRARY_API_KEY", func() string { return config.NationalLibraryAPIKey }},
	{"PIXABAY_API_KEY", func() string { return config.PixabayAPIKey }},
	{"MCP_AUTH_TOKEN", func() string { return config.MCPAuthToken }},
}

func maskKey(v string) string {
	if len(v) == 0 {
		return "(미설정)"
	}
	if len(v) <= 8 {
		return "****"
	}
	return v[:4] + "****" + v[len(v)-4:]
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "설정 관리",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "API 키 현황 출력 (마스킹)",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("API 키 현황:")
		for _, k := range apiKeys {
			fmt.Printf("  %-30s %s\n", k.label, maskKey(k.getter()))
		}
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "미설정 API 키 경고",
	Run: func(cmd *cobra.Command, args []string) {
		missing := []string{}
		for _, k := range apiKeys {
			if k.getter() == "" {
				missing = append(missing, k.label)
			}
		}
		if len(missing) == 0 {
			fmt.Println("모든 API 키가 설정되어 있습니다.")
			return
		}
		fmt.Printf("미설정 API 키 (%d개):\n", len(missing))
		for _, label := range missing {
			fmt.Printf("  - %s\n", label)
		}
	},
}

var configGenerateTokenCmd = &cobra.Command{
	Use:   "generate-token",
	Short: "serve http용 MCP_AUTH_TOKEN 무작위 생성",
	RunE: func(cmd *cobra.Command, args []string) error {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			return fmt.Errorf("난수 생성 실패: %w", err)
		}
		fmt.Println(hex.EncodeToString(buf))
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd, configValidateCmd, configGenerateTokenCmd)
	RootCmd.AddCommand(configCmd)
}
