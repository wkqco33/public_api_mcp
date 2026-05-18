package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"wkqcosoft.com/public-api-mcp/config"
	"wkqcosoft.com/public-api-mcp/httpclient"
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
	}, getFuelAvgPrice)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_fuel_sido_price",
		Description: "시도별 주유소 평균가격 조회 (오피넷).",
	}, getFuelSidoPrice)
}

func opinetGet(endpoint string, params map[string]string) (map[string]any, error) {
	if config.OpinetAPIKey == "" {
		return nil, fmt.Errorf("OPINET_API_KEY가 설정되지 않았습니다")
	}
	req, _ := http.NewRequest("GET", "https://www.opinet.co.kr/api/"+endpoint, nil)
	q := req.URL.Query()
	q.Set("code", config.OpinetAPIKey)
	q.Set("out", "json")
	for k, v := range params {
		q.Set(k, v)
	}
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

func getFuelAvgPrice(_ context.Context, _ *mcp.CallToolRequest, in fuelAvgInput) (*mcp.CallToolResult, any, error) {
	params := map[string]string{}
	if in.Prodcd != "" {
		params["prodcd"] = in.Prodcd
	}
	data, err := opinetGet("avgAllPrice.do", params)
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
		price, _ := item["PRICE"].(float64)
		fmt.Fprintf(&sb, "  %s: %.2f원/L (%s원)\n", name, price, diffStr(item["DIFF"]))
	}
	return toolText(sb.String()), nil, nil
}

type fuelSidoInput struct {
	Sido   string `json:"sido,omitempty" jsonschema:"시도코드 2자리 (생략 시 전체)"`
	Prodcd string `json:"prodcd,omitempty" jsonschema:"제품코드 (생략 시 전체)"`
}

func getFuelSidoPrice(_ context.Context, _ *mcp.CallToolRequest, in fuelSidoInput) (*mcp.CallToolResult, any, error) {
	params := map[string]string{}
	if in.Sido != "" {
		params["sido"] = in.Sido
	}
	if in.Prodcd != "" {
		params["prodcd"] = in.Prodcd
	}
	data, err := opinetGet("avgSidoPrice.do", params)
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
		price, _ := item["PRICE"].(float64)
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
