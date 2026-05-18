package tools

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"wkqcosoft.com/public-api-mcp/config"
	"wkqcosoft.com/public-api-mcp/httpclient"
)

var realEstateEndpoints = map[string]string{
	"apt_trade": "http://apis.data.go.kr/1613000/RTMSDataSvcAptTrade/getRTMSDataSvcAptTrade",
	"apt_rent":  "http://apis.data.go.kr/1613000/RTMSDataSvcAptRent/getRTMSDataSvcAptRent",
	"rh_trade":  "http://apis.data.go.kr/1613000/RTMSDataSvcRHTrade/getRTMSDataSvcRHTrade",
	"rh_rent":   "http://apis.data.go.kr/1613000/RTMSDataSvcRHRent/getRTMSDataSvcRHRent",
}

var realEstateTitles = map[string]string{
	"apt_trade": "아파트 매매",
	"apt_rent":  "아파트 전월세",
	"rh_trade":  "연립다세대 매매",
	"rh_rent":   "연립다세대 전월세",
}

type realEstateInput struct {
	APIType   string `json:"api_type" jsonschema:"거래 유형: apt_trade(아파트매매), apt_rent(아파트전세), rh_trade(연립매매), rh_rent(연립전세)"`
	LawdCd    string `json:"lawd_cd" jsonschema:"지역코드 5자리 (예: 11110 = 서울 종로구)"`
	DealYmd   string `json:"deal_ymd" jsonschema:"계약년월 (YYYYMM, 예: 202405)"`
	NumOfRows int    `json:"num_of_rows,omitempty" jsonschema:"결과 수 (기본: 10)"`
}

type xmlItem struct {
	Fields []xmlField `xml:",any"`
}

type xmlField struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type xmlResponse struct {
	Items []xmlItem `xml:"body>items>item"`
}

func RegisterRealEstate(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_real_estate_data",
		Description: "부동산 실거래가 정보 조회. 아파트/연립다세대 매매·전월세 거래 내역 조회.",
	}, getRealEstateData)
}

func getRealEstateData(_ context.Context, _ *mcp.CallToolRequest, in realEstateInput) (*mcp.CallToolResult, any, error) {
	if config.PublicDataPortalAPIKey == "" {
		return toolError("PUBLIC_DATA_PORTAL_API_KEY가 설정되지 않았습니다"), nil, nil
	}

	endpoint, ok := realEstateEndpoints[in.APIType]
	if !ok {
		return toolError("유효하지 않은 API 타입: %s (apt_trade, apt_rent, rh_trade, rh_rent 중 하나)", in.APIType), nil, nil
	}

	numOfRows := in.NumOfRows
	if numOfRows <= 0 {
		numOfRows = 10
	}

	req, _ := http.NewRequest("GET", endpoint, nil)
	q := req.URL.Query()
	q.Set("serviceKey", config.PublicDataPortalAPIKey)
	q.Set("LAWD_CD", in.LawdCd)
	q.Set("DEAL_YMD", in.DealYmd)
	q.Set("pageNo", "1")
	q.Set("numOfRows", fmt.Sprintf("%d", numOfRows))
	req.URL.RawQuery = q.Encode()

	res, err := httpclient.Default.Do(req)
	if err != nil {
		return toolError("API 요청 실패: %v", err), nil, nil
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 5*1024*1024))
	if err != nil {
		return toolError("응답 읽기 실패: %v", err), nil, nil
	}

	var xmlRes xmlResponse
	if err := xml.Unmarshal(body, &xmlRes); err != nil {
		return toolError("XML 파싱 실패: %v", err), nil, nil
	}
	if len(xmlRes.Items) == 0 {
		return toolText(fmt.Sprintf("[%s] 해당 지역의 데이터가 없습니다.", in.DealYmd)), nil, nil
	}

	title := realEstateTitles[in.APIType]
	var sb strings.Builder
	fmt.Fprintf(&sb, "### %s 실거래가 (%s, %s) ###\n\n", title, in.LawdCd, in.DealYmd)

	for i, item := range xmlRes.Items {
		fmt.Fprintf(&sb, "[%d]\n", i+1)
		for _, field := range item.Fields {
			if field.Value != "" {
				fmt.Fprintf(&sb, "  %s: %s\n", field.XMLName.Local, strings.TrimSpace(field.Value))
			}
		}
		sb.WriteString("\n")
	}
	return toolText(sb.String()), nil, nil
}
