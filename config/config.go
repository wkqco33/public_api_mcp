package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

var (
	KakaoRESTAPIKey string

	ExchangeAPIKey string

	PublicDataPortalAPIKey string
	SeoulOpenAPIKey        string

	KMAWeatherAPIKey string

	OpenDARTAPIKey string

	OpinetAPIKey string

	NationalLibraryAPIKey string

	PixabayAPIKey string

	// MCPAuthToken은 serve http(SSE) 진입점을 보호하는 공유 비밀키다.
	// 비어 있으면 serve http는 시작을 거부한다(--insecure로만 예외 허용).
	MCPAuthToken string
)

// Load는 envFile을 로딩한 뒤 모든 config 변수를 채운다.
// envFile이 빈 문자열이면 ".env"를 시도한다 (없으면 무시).
func Load(envFile string) error {
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			return fmt.Errorf("env 파일 로드 실패: %w", err)
		}
	} else if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf(".env 파일 로드 실패: %w", err)
	}

	KakaoRESTAPIKey = os.Getenv("KAKAO_REST_API_KEY")

	ExchangeAPIKey = os.Getenv("EXCHANGE_API_KEY")

	PublicDataPortalAPIKey = os.Getenv("PUBLIC_DATA_PORTAL_API_KEY")
	SeoulOpenAPIKey = os.Getenv("SEOUL_OPEN_API_KEY")

	KMAWeatherAPIKey = os.Getenv("KMA_WEATHER_API_KEY")

	OpenDARTAPIKey = os.Getenv("OPENDART_API_KEY")

	OpinetAPIKey = os.Getenv("OPINET_API_KEY")

	NationalLibraryAPIKey = os.Getenv("NATIONAL_LIBRARY_API_KEY")

	PixabayAPIKey = os.Getenv("PIXABAY_API_KEY")

	MCPAuthToken = os.Getenv("MCP_AUTH_TOKEN")
	return nil
}
