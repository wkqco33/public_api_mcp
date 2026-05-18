package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"wkqcosoft.com/public-api-mcp/cache"
	"wkqcosoft.com/public-api-mcp/config"
	"wkqcosoft.com/public-api-mcp/httpclient"
)

var exchangeCache = cache.New[string, string]()

type exchangeInput struct {
	Date string `json:"date,omitempty" jsonschema:"조회 날짜 (YYYYMMDD 형식, 예: 20240307). 생략 시 최근 영업일 데이터"`
}

func RegisterExchange(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_exchange_rates",
		Description: "한국수출입은행 API를 통해 최신 환율 정보 조회. 매매기준율, 송금 환율 등을 확인.",
	}, getExchangeRates)
}

func getExchangeRates(_ context.Context, _ *mcp.CallToolRequest, in exchangeInput) (*mcp.CallToolResult, any, error) {
	if config.ExchangeAPIKey == "" {
		return toolError("EXCHANGE_API_KEY가 설정되지 않았습니다"), nil, nil
	}

	kst := time.FixedZone("KST", 9*60*60)
	date := in.Date
	if date == "" {
		date = time.Now().In(kst).Format("20060102")
	}

	if cached, ok := exchangeCache.Get(date); ok {
		slog.Debug("환율 캐시 hit", "date", date)
		return toolText(cached), nil, nil
	}
	slog.Debug("환율 API 조회", "date", date)

	result, err := fetchExchangeDate(date, config.ExchangeAPIKey, kst)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	exchangeCache.Set(date, result, time.Hour)
	return toolText(result), nil, nil
}

func fetchExchangeDate(date, apiKey string, kst *time.Location) (string, error) {
	data, err := callExchangeAPI(date, apiKey)
	if err != nil {
		return "", err
	}

	if len(data) == 0 {
		// 비영업일이면 최대 7일 이전까지 탐색
		current, _ := time.ParseInLocation("20060102", date, kst)
		for i := 1; i <= 7; i++ {
			prev := current.AddDate(0, 0, -i).Format("20060102")
			data, err = callExchangeAPI(prev, apiKey)
			if err != nil {
				continue
			}
			if len(data) > 0 {
				date = prev
				break
			}
		}
	}
	if len(data) == 0 {
		return "", fmt.Errorf("환율 데이터를 찾을 수 없습니다 (기준일: %s)", date)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "### 환율 정보 (기준일: %s)\n", date)
	for _, item := range data {
		unit, _ := item["cur_unit"].(string)
		name, _ := item["cur_nm"].(string)
		baseRate, _ := item["deal_bas_r"].(string)
		ttb, _ := item["ttb"].(string)
		tts, _ := item["tts"].(string)
		fmt.Fprintf(&sb, "- **%s (%s)**: %s원 (송금받을때: %s, 보낼때: %s)\n", name, unit, baseRate, ttb, tts)
	}
	return sb.String(), nil
}

func callExchangeAPI(date, apiKey string) ([]map[string]any, error) {
	req, _ := http.NewRequest("GET", "https://oapi.koreaexim.go.kr/site/program/financial/exchangeJSON", nil)
	q := req.URL.Query()
	q.Set("authkey", apiKey)
	q.Set("searchdate", date)
	q.Set("data", "AP01")
	req.URL.RawQuery = q.Encode()

	res, err := httpclient.Default.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 요청 실패: %v", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %v", err)
	}

	var data []map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %v", err)
	}
	return data, nil
}
