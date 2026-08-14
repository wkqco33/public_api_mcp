package tools

import (
	"context"
	"encoding/json"
	"fmt"
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

type ExchangeHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type ExchangeClient struct {
	httpClient ExchangeHTTPDoer
	baseURL    string
	apiKey     string
}

func NewExchangeClient(client ExchangeHTTPDoer, baseURL, apiKey string) *ExchangeClient {
	return &ExchangeClient{httpClient: client, baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey}
}

type ExchangeClock interface{ Now() time.Time }
type systemExchangeClock struct{}

func (systemExchangeClock) Now() time.Time { return time.Now() }

func defaultExchangeClient() *ExchangeClient {
	return NewExchangeClient(httpclient.Default, "https://oapi.koreaexim.go.kr", config.ExchangeAPIKey)
}

type exchangeRate struct {
	Unit     string `json:"cur_unit"`
	Name     string `json:"cur_nm"`
	BaseRate string `json:"deal_bas_r"`
	TTB      string `json:"ttb"`
	TTS      string `json:"tts"`
}

type exchangeInput struct {
	Date string `json:"date,omitempty" jsonschema:"조회 날짜 (YYYYMMDD 형식, 예: 20240307). 생략 시 최근 영업일 데이터"`
}

func RegisterExchange(s *mcp.Server) {
	RegisterExchangeWithDependencies(s, defaultExchangeClient(), systemExchangeClock{})
}

func RegisterExchangeWithDependencies(s *mcp.Server, client *ExchangeClient, clock ExchangeClock) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_exchange_rates",
		Description: "한국수출입은행 API를 통해 최신 환율 정보 조회. 매매기준율, 송금 환율 등을 확인.",
	}, withRecover("get_exchange_rates", func(ctx context.Context, req *mcp.CallToolRequest, in exchangeInput) (*mcp.CallToolResult, any, error) {
		return getExchangeRatesWithDependencies(ctx, client, clock, in)
	}))
}

func getExchangeRates(ctx context.Context, _ *mcp.CallToolRequest, in exchangeInput) (*mcp.CallToolResult, any, error) {
	return getExchangeRatesWithDependencies(ctx, defaultExchangeClient(), systemExchangeClock{}, in)
}

func getExchangeRatesWithDependencies(ctx context.Context, client *ExchangeClient, clock ExchangeClock, in exchangeInput) (*mcp.CallToolResult, any, error) {
	if client.apiKey == "" {
		return toolError("EXCHANGE_API_KEY가 설정되지 않았습니다"), nil, nil
	}

	kst := time.FixedZone("KST", 9*60*60)
	date := in.Date
	if date == "" {
		date = clock.Now().In(kst).Format("20060102")
	}

	if cached, ok := exchangeCache.Get(date); ok {
		slog.Debug("환율 캐시 hit", "date", date)
		return toolText(cached), nil, nil
	}
	slog.Debug("환율 API 조회", "date", date)

	result, err := fetchExchangeDateWithClient(ctx, client, date, kst)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	exchangeCache.Set(date, result, time.Hour)
	return toolText(result), nil, nil
}

func fetchExchangeDate(ctx context.Context, date, apiKey string, kst *time.Location) (string, error) {
	return fetchExchangeDateWithClient(ctx, NewExchangeClient(httpclient.Default, "https://oapi.koreaexim.go.kr", apiKey), date, kst)
}

func fetchExchangeDateWithClient(ctx context.Context, client *ExchangeClient, date string, kst *time.Location) (string, error) {
	data, err := client.call(ctx, date)
	if err != nil {
		return "", err
	}

	if len(data) == 0 {
		// 비영업일이면 최대 7일 이전까지 탐색
		current, _ := time.ParseInLocation("20060102", date, kst)
		for i := 1; i <= 7; i++ {
			prev := current.AddDate(0, 0, -i).Format("20060102")
			data, err = client.call(ctx, prev)
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
		fmt.Fprintf(&sb, "- **%s (%s)**: %s원 (송금받을때: %s, 보낼때: %s)\n", item.Name, item.Unit, item.BaseRate, item.TTB, item.TTS)
	}
	return sb.String(), nil
}

func callExchangeAPI(ctx context.Context, date, apiKey string) ([]exchangeRate, error) {
	return NewExchangeClient(httpclient.Default, "https://oapi.koreaexim.go.kr", apiKey).call(ctx, date)
}

func (c *ExchangeClient) call(ctx context.Context, date string) ([]exchangeRate, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/site/program/financial/exchangeJSON", nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}
	q := req.URL.Query()
	q.Set("authkey", c.apiKey)
	q.Set("searchdate", date)
	q.Set("data", "AP01")
	req.URL.RawQuery = q.Encode()

	body, _, err := httpclient.FetchBodyWithClient(c.httpClient, req, maxAPIBodyBytes)
	if err != nil {
		return nil, err
	}

	var data []exchangeRate
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %v", err)
	}
	return data, nil
}
