package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"wkqcosoft.com/public-api-mcp/config"
	"wkqcosoft.com/public-api-mcp/httpclient"
)

type busStopSearchInput struct {
	StopName string `json:"stop_name" jsonschema:"정류소 이름 (예: 강남역, 서울역)"`
	CityCode string `json:"city_code,omitempty" jsonschema:"도시코드: 11=서울, 21=부산, 31=경기 (기본: 11)"`
}

type busArrivalInput struct {
	NodeID   string `json:"node_id" jsonschema:"정류소 ID (search_bus_stop으로 확인)"`
	CityCode string `json:"city_code,omitempty" jsonschema:"도시코드: 11=서울, 21=부산, 31=경기 (기본: 11)"`
}

type subwayArrivalInput struct {
	StationName string `json:"station_name" jsonschema:"지하철역 이름 (예: 강남, 서울)"`
}

func RegisterTransport(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_bus_stop",
		Description: "버스 정류소 이름으로 정류소 ID 검색. get_bus_arrival 사용 전에 ID를 확인하는 데 사용.",
	}, withRecover("search_bus_stop", searchBusStop))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_bus_arrival",
		Description: "버스 정류소의 도착 예정 버스 정보 조회. 먼저 search_bus_stop으로 정류소 ID 확인 필요.",
	}, withRecover("get_bus_arrival", getBusArrival))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_subway_arrival",
		Description: "서울 지하철역의 실시간 도착 예정 열차 정보 조회.",
	}, withRecover("get_subway_arrival", getSubwayArrival))
}

func busAPIGet(ctx context.Context, path string, params map[string]string) ([]any, error) {
	if config.PublicDataPortalAPIKey == "" {
		return nil, fmt.Errorf("PUBLIC_DATA_PORTAL_API_KEY가 설정되지 않았습니다")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://apis.data.go.kr/1613000/"+path, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}
	q := req.URL.Query()
	q.Set("serviceKey", config.PublicDataPortalAPIKey)
	q.Set("numOfRows", "10")
	q.Set("pageNo", "1")
	q.Set("_type", "json")
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	body, _, err := httpclient.FetchBody(req, maxAPIBodyBytes)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %v", err)
	}
	response, _ := data["response"].(map[string]any)
	respBody, _ := response["body"].(map[string]any)
	items_, _ := respBody["items"].(map[string]any)
	items, _ := items_["item"].([]any)
	if items == nil {
		// 단일 아이템인 경우
		if single, ok := items_["item"].(map[string]any); ok {
			items = []any{single}
		}
	}
	return items, nil
}

func searchBusStop(ctx context.Context, _ *mcp.CallToolRequest, in busStopSearchInput) (*mcp.CallToolResult, any, error) {
	cityCode := in.CityCode
	if cityCode == "" {
		cityCode = "11"
	}
	items, err := busAPIGet(ctx, "BusSttnInfoInqireService/getSttnNoList", map[string]string{
		"cityCode": cityCode,
		"nodeNm":   in.StopName,
	})
	if err != nil {
		return toolError("%v", err), nil, nil
	}
	if len(items) == 0 {
		return toolText(fmt.Sprintf("'%s' 정류소를 찾을 수 없습니다", in.StopName)), nil, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "## '%s' 정류소 검색 결과\n", in.StopName)
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		nodeID, _ := item["nodeid"].(string)
		nodeNm, _ := item["nodenm"].(string)
		fmt.Fprintf(&sb, "- %s (ID: %s)\n", nodeNm, nodeID)
	}
	sb.WriteString("\n→ 도착 정보는 get_bus_arrival(node_id) 도구를 사용하세요.")
	return toolText(sb.String()), nil, nil
}

func getBusArrival(ctx context.Context, _ *mcp.CallToolRequest, in busArrivalInput) (*mcp.CallToolResult, any, error) {
	cityCode := in.CityCode
	if cityCode == "" {
		cityCode = "11"
	}
	items, err := busAPIGet(ctx, "ArvlInfoInqireForOrganicVehicle/getSttnAcctoArvlPrearngeInfoList", map[string]string{
		"cityCode": cityCode,
		"nodeId":   in.NodeID,
	})
	if err != nil {
		return toolError("%v", err), nil, nil
	}
	if len(items) == 0 {
		return toolText(fmt.Sprintf("정류소 '%s'의 도착 예정 버스 정보 없음", in.NodeID)), nil, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "## 정류소 %s 도착 예정\n", in.NodeID)
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		routeNo, _ := item["routeno"].(string)
		arrCount, _ := item["arrprevstationcnt"].(float64)
		arrTime, _ := item["arrtime"].(float64)
		arrStr := "곧 도착"
		if arrTime > 0 {
			arrStr = fmt.Sprintf("%.0f분", arrTime/60)
		}
		fmt.Fprintf(&sb, "- %s번 버스: %s 후 도착 (전 %.0f정류장)\n", routeNo, arrStr, arrCount)
	}
	return toolText(sb.String()), nil, nil
}

func getSubwayArrival(ctx context.Context, _ *mcp.CallToolRequest, in subwayArrivalInput) (*mcp.CallToolResult, any, error) {
	if config.SeoulOpenAPIKey == "" {
		return toolError("SEOUL_OPEN_API_KEY가 설정되지 않았습니다"), nil, nil
	}

	url := fmt.Sprintf("http://swopenapi.seoul.go.kr/api/subway/%s/json/realtimeStationArrival/0/10/%s",
		config.SeoulOpenAPIKey, in.StationName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return toolError("요청 생성 실패: %v", err), nil, nil
	}
	body, _, err := httpclient.FetchBody(req, maxAPIBodyBytes)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return toolError("응답 파싱 실패: %v", err), nil, nil
	}

	arrivals, _ := data["realtimeArrivalList"].([]any)
	if len(arrivals) == 0 {
		return toolText(fmt.Sprintf("'%s' 역 도착 정보 없음", in.StationName)), nil, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s역 실시간 도착 정보\n", in.StationName)
	for _, raw := range arrivals {
		a, _ := raw.(map[string]any)
		lineID, _ := a["subwayId"].(string)
		trainLine, _ := a["trainLineNm"].(string)
		msg, _ := a["arvlMsg2"].(string)
		dest, _ := a["bstatnNm"].(string)
		fmt.Fprintf(&sb, "- [%s호선] %s → %s: %s\n", lineID, trainLine, dest, msg)
	}
	return toolText(sb.String()), nil, nil
}
