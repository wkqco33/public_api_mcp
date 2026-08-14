package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/wkqco33/public_api_mcp/config"
	"github.com/wkqco33/public_api_mcp/httpclient"
)

var prodNames = map[string]string{
	"B027": "보통휘발유",
	"D047": "자동차경유",
	"B034": "고급휘발유",
	"C004": "실내등유",
	"K015": "자동차부탄",
}

type fuelAvgInput struct {
	Prodcd string `json:"prodcd,omitempty" jsonschema:"제품코드: B027(보통휘발유), D047(경유), B034(고급휘발유), C004(등유). 생략 시 전체"`
}

func RegisterFuel(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_fuel_avg_price",
		Description: "전국 현재 주유소 평균가격 조회 (오피넷). prodcd 미입력 시 모든 제품 조회.",
	}, withRecover("get_fuel_avg_price", getFuelAvgPrice))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_fuel_sido_price",
		Description: "시도별 주유소 평균가격 조회 (오피넷).",
	}, withRecover("get_fuel_sido_price", getFuelSidoPrice))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_fuel_low_top10",
		Description: "특정 지역의 제품별 최저가 주유소 Top 10 목록 조회 (오피넷).",
	}, withRecover("get_fuel_low_top10", getFuelLowTop10))
}

type FuelHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}
type opinetClient struct {
	httpClient FuelHTTPDoer
	baseURL    string
	apiKey     string
}

func NewOpinetClient(client FuelHTTPDoer, baseURL, apiKey string) *opinetClient {
	return &opinetClient{httpClient: client, baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey}
}
func defaultOpinetClient() *opinetClient {
	return NewOpinetClient(httpclient.Default, "https://www.opinet.co.kr/api", config.OpinetAPIKey)
}

func opinetGet(ctx context.Context, endpoint string, params map[string]string) (map[string]any, error) {
	return opinetGetWithClient(ctx, defaultOpinetClient(), endpoint, params)
}
func opinetGetWithClient(ctx context.Context, client *opinetClient, endpoint string, params map[string]string) (map[string]any, error) {
	if client.apiKey == "" {
		return nil, fmt.Errorf("OPINET_API_KEY가 설정되지 않았습니다")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", client.baseURL+"/"+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}
	q := req.URL.Query()
	q.Set("code", client.apiKey)
	q.Set("out", "json")
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	body, _, err := httpclient.FetchBodyWithClient(client.httpClient, req, maxAPIBodyBytes)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %v", err)
	}
	return data, nil
}

func diffStr(diff any) string {
	var v float64
	switch d := diff.(type) {
	case float64:
		v = d
	case string:
		f, err := strconv.ParseFloat(d, 64)
		if err != nil {
			return d
		}
		v = f
	default:
		return "→0"
	}
	if v > 0 {
		return fmt.Sprintf("▲%.1f", v)
	} else if v < 0 {
		return fmt.Sprintf("▼%.1f", -v)
	}
	return "→0"
}

func parseFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

func getFuelAvgPrice(ctx context.Context, _ *mcp.CallToolRequest, in fuelAvgInput) (*mcp.CallToolResult, any, error) {
	return getFuelAvgPriceWithClient(ctx, defaultOpinetClient(), in)
}
func getFuelAvgPriceWithClient(ctx context.Context, client *opinetClient, in fuelAvgInput) (*mcp.CallToolResult, any, error) {
	params := map[string]string{}
	if in.Prodcd != "" {
		params["prodcd"] = in.Prodcd
	}
	data, err := opinetGetWithClient(ctx, client, "avgAllPrice.do", params)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	result, _ := data["RESULT"].(map[string]any)
	oils, _ := result["OIL"].([]any)
	if len(oils) == 0 {
		return toolText("조회 결과가 없습니다"), nil, nil
	}

	var sb strings.Builder
	first, _ := oils[0].(map[string]any)
	tradeDt, _ := first["TRADE_DT"].(string)
	fmt.Fprintf(&sb, "[%s 기준] 전국 평균가격\n", tradeDt)
	for _, raw := range oils {
		item, _ := raw.(map[string]any)
		prodcd, _ := item["PRODCD"].(string)
		prodnm, _ := item["PRODNM"].(string)
		name := prodNames[prodcd]
		if name == "" {
			name = prodnm
		}
		price := parseFloat(item["PRICE"])
		fmt.Fprintf(&sb, "  %s: %.2f원/L (%s원)\n", name, price, diffStr(item["DIFF"]))
	}
	return toolText(sb.String()), nil, nil
}

type fuelSidoInput struct {
	Sido   string `json:"sido,omitempty" jsonschema:"시도코드 2자리 (생략 시 전체)"`
	Prodcd string `json:"prodcd,omitempty" jsonschema:"제품코드 (생략 시 전체)"`
}

func getFuelSidoPrice(ctx context.Context, _ *mcp.CallToolRequest, in fuelSidoInput) (*mcp.CallToolResult, any, error) {
	return getFuelSidoPriceWithClient(ctx, defaultOpinetClient(), in)
}
func getFuelSidoPriceWithClient(ctx context.Context, client *opinetClient, in fuelSidoInput) (*mcp.CallToolResult, any, error) {
	params := map[string]string{}
	if in.Sido != "" {
		params["sido"] = in.Sido
	}
	if in.Prodcd != "" {
		params["prodcd"] = in.Prodcd
	}
	data, err := opinetGetWithClient(ctx, client, "avgSidoPrice.do", params)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	result, _ := data["RESULT"].(map[string]any)
	oils, _ := result["OIL"].([]any)
	if len(oils) == 0 {
		return toolText("조회 결과가 없습니다"), nil, nil
	}

	sidoMap := map[string][]string{}
	var sidoOrder []string
	for _, raw := range oils {
		item, _ := raw.(map[string]any)
		sname, _ := item["SIDONM"].(string)
		prodcd, _ := item["PRODCD"].(string)
		name := prodNames[prodcd]
		if name == "" {
			name = prodcd
		}
		price := parseFloat(item["PRICE"])
		line := fmt.Sprintf("%s: %.2f원/L (%s원)", name, price, diffStr(item["DIFF"]))
		if _, exists := sidoMap[sname]; !exists {
			sidoOrder = append(sidoOrder, sname)
		}
		sidoMap[sname] = append(sidoMap[sname], line)
	}

	var sb strings.Builder
	sb.WriteString("시도별 평균가격\n")
	for _, sname := range sidoOrder {
		fmt.Fprintf(&sb, "\n[%s]\n", sname)
		for _, p := range sidoMap[sname] {
			fmt.Fprintf(&sb, "  %s\n", p)
		}
	}
	return toolText(sb.String()), nil, nil
}

var pollNames = map[string]string{
	"SKE": "SK에너지",
	"GSC": "GS칼텍스",
	"HDO": "현대오일뱅크",
	"SOL": "S-OIL",
	"RTO": "알뜰주유소",
	"RTX": "고속도로알뜰",
	"NHO": "NH알뜰",
	"E10": "E1",
	"SKG": "SK가스",
}

type fuelLowTop10Input struct {
	Prodcd string `json:"prodcd" jsonschema:"제품코드: B027(보통휘발유), D047(경유), B034(고급휘발유), C004(등유), K015(부탄)"`
	Area   string `json:"area" jsonschema:"지역코드 (예: 서울 01, 서울종로 0101, 경기수원 0202 등)"`
}

func getFuelLowTop10(ctx context.Context, _ *mcp.CallToolRequest, in fuelLowTop10Input) (*mcp.CallToolResult, any, error) {
	return getFuelLowTop10WithClient(ctx, defaultOpinetClient(), in)
}
func getFuelLowTop10WithClient(ctx context.Context, client *opinetClient, in fuelLowTop10Input) (*mcp.CallToolResult, any, error) {
	prod := strings.TrimSpace(in.Prodcd)
	if prod == "" {
		return toolError("제품코드(prodcd)는 필수 입력 항목입니다 (예: B027)"), nil, nil
	}

	area := strings.TrimSpace(in.Area)
	if area == "" {
		return toolError("지역코드(area)는 필수 입력 항목입니다 (예: 서울 종로 0101)"), nil, nil
	}

	params := map[string]string{
		"prodcd": prod,
		"area":   area,
	}

	data, err := opinetGetWithClient(ctx, client, "lowTop10.do", params)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	result, _ := data["RESULT"].(map[string]any)
	oils, _ := result["OIL"].([]any)
	if len(oils) == 0 {
		return toolText("조회 결과가 없습니다. 제품코드 또는 지역코드를 확인해 주세요."), nil, nil
	}

	var sb strings.Builder
	prodName := prodNames[prod]
	if prodName == "" {
		prodName = prod
	}
	fmt.Fprintf(&sb, "### 지역별 %s 최저가 주유소 Top 10\n\n", prodName)
	fmt.Fprintln(&sb, "| 순위 | 주유소명 | 상표 | 가격 | 주소 |")
	fmt.Fprintln(&sb, "|---|---|---|---|---|")

	for i, raw := range oils {
		item, _ := raw.(map[string]any)
		osNm, _ := item["OS_NM"].(string)
		pollCd, _ := item["POLL_DIV_CD"].(string)
		pollName := pollNames[pollCd]
		if pollName == "" {
			pollName = pollCd
		}
		price := parseFloat(item["PRICE"])
		adr, _ := item["NEW_ADR"].(string)
		if adr == "" {
			adr, _ = item["VAN_ADR"].(string)
		}

		fmt.Fprintf(&sb, "| %d | %s | %s | %.2f원 | %s |\n", i+1, osNm, pollName, price, adr)
	}

	return toolText(sb.String()), nil, nil
}
