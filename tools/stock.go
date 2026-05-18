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

var stockCache = cache.New[string, string]()

type stockPriceInput struct {
	ItemName       string `json:"item_name,omitempty" jsonschema:"종목명 (예: 삼성전자)"`
	BaseDate       string `json:"base_date,omitempty" jsonschema:"기준일자 (YYYYMMDD)"`
	BeginDate      string `json:"begin_date,omitempty" jsonschema:"시작일자 (YYYYMMDD)"`
	EndDate        string `json:"end_date,omitempty" jsonschema:"종료일자 (YYYYMMDD)"`
	MarketCategory string `json:"market_category,omitempty" jsonschema:"시장구분: KOSPI, KOSDAQ, KONEX"`
	NumOfRows      int    `json:"num_of_rows,omitempty" jsonschema:"결과 수 (기본: 10)"`
}

type stockAnalysisInput struct {
	ItemName string `json:"item_name" jsonschema:"종목명 (예: 삼성전자)"`
}

func RegisterStock(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_stock_price",
		Description: "공공데이터포털에서 주식 시세 조회 (종가, 등락, 거래량 등).",
	}, getStockPrice)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_stock_analysis_data",
		Description: "특정 종목의 최근 30일 시세 + 관련 뉴스를 종합하여 분석 데이터 제공.",
	}, getStockAnalysisData)
}

func getStockPrice(_ context.Context, _ *mcp.CallToolRequest, in stockPriceInput) (*mcp.CallToolResult, any, error) {
	if config.PublicDataPortalAPIKey == "" {
		return toolError("PUBLIC_DATA_PORTAL_API_KEY가 설정되지 않았습니다"), nil, nil
	}

	cacheKey := fmt.Sprintf("%s:%s:%s:%s", in.ItemName, in.BaseDate, in.BeginDate, in.EndDate)
	if cached, ok := stockCache.Get(cacheKey); ok {
		slog.Debug("주식 캐시 hit", "key", cacheKey)
		return toolText(cached), nil, nil
	}
	slog.Debug("주식 API 조회", "item", in.ItemName)

	numOfRows := in.NumOfRows
	if numOfRows <= 0 {
		numOfRows = 10
	}

	req, _ := http.NewRequest("GET", "https://apis.data.go.kr/1160100/service/GetStockSecuritiesInfoService/getStockPriceInfo", nil)
	q := req.URL.Query()
	q.Set("serviceKey", config.PublicDataPortalAPIKey)
	q.Set("numOfRows", fmt.Sprintf("%d", numOfRows))
	q.Set("pageNo", "1")
	q.Set("resultType", "json")
	if in.ItemName != "" {
		q.Set("itmsNm", in.ItemName)
	}
	if in.BaseDate != "" {
		q.Set("basDt", in.BaseDate)
	}
	if in.BeginDate != "" {
		q.Set("beginBasDt", in.BeginDate)
	}
	if in.EndDate != "" {
		q.Set("endBasDt", in.EndDate)
	}
	if in.MarketCategory != "" {
		q.Set("mrktCls", in.MarketCategory)
	}
	req.URL.RawQuery = q.Encode()

	res, err := httpclient.Default.Do(req)
	if err != nil {
		return toolError("API 요청 실패: %v", err), nil, nil
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return toolError("응답 파싱 실패: %v", err), nil, nil
	}

	response, _ := data["response"].(map[string]any)
	header, _ := response["header"].(map[string]any)
	if code, _ := header["resultCode"].(string); code != "00" {
		msg, _ := header["resultMsg"].(string)
		return toolError("API 오류: %s", msg), nil, nil
	}

	respBody, _ := response["body"].(map[string]any)
	items_, _ := respBody["items"].(map[string]any)
	itemList, _ := items_["item"].([]any)

	if len(itemList) == 0 {
		return toolText("조회 결과가 없습니다"), nil, nil
	}

	var sb strings.Builder
	for _, raw := range itemList {
		item, _ := raw.(map[string]any)
		basDt, _ := item["basDt"].(string)
		itmsNm, _ := item["itmsNm"].(string)
		srtnCd, _ := item["srtnCd"].(string)
		clpr, _ := item["clpr"].(string)
		vs, _ := item["vs"].(string)
		fltRt, _ := item["fltRt"].(string)
		trqu, _ := item["trqu"].(string)
		fmt.Fprintf(&sb, "[%s] %s (%s) | 종가: %s (대비 %s, %s%%) | 거래량: %s\n",
			basDt, itmsNm, srtnCd, clpr, vs, fltRt, trqu)
	}

	result := sb.String()
	stockCache.Set(cacheKey, result, 5*time.Minute)
	return toolText(result), nil, nil
}

func getStockAnalysisData(_ context.Context, _ *mcp.CallToolRequest, in stockAnalysisInput) (*mcp.CallToolResult, any, error) {
	kst := time.FixedZone("KST", 9*60*60)
	now := time.Now().In(kst)
	begin := now.AddDate(0, 0, -30).Format("20060102")
	end := now.Format("20060102")

	priceResult, _, _ := getStockPrice(context.Background(), nil, stockPriceInput{
		ItemName:  in.ItemName,
		BeginDate: begin,
		EndDate:   end,
		NumOfRows: 30,
	})
	priceText := ""
	if priceResult != nil && len(priceResult.Content) > 0 {
		if tc, ok := priceResult.Content[0].(*mcp.TextContent); ok {
			priceText = tc.Text
		}
	}

	newsResult, _, _ := getNews(context.Background(), nil, getNewsInput{Topic: in.ItemName})
	newsText := ""
	if newsResult != nil && len(newsResult.Content) > 0 {
		if tc, ok := newsResult.Content[0].(*mcp.TextContent); ok {
			newsText = tc.Text
		}
	}

	result := fmt.Sprintf("### [%s] 분석 데이터 ###\n\n1. 최근 30일 시세 추이:\n%s\n\n2. 관련 최신 뉴스:\n%s",
		in.ItemName, priceText, newsText)
	return toolText(result), nil, nil
}
