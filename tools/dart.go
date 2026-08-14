package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"wkqcosoft.com/public-api-mcp/cache"
	"wkqcosoft.com/public-api-mcp/config"
	"wkqcosoft.com/public-api-mcp/httpclient"
)

const dartCacheTTL = 7 * 24 * time.Hour

var (
	dartCorpCodes   map[string]string
	dartCorpCodesMu sync.RWMutex
	corpCodeCache   = cache.New[string, string]()
)

type DARTHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}
type DARTClient struct {
	httpClient DARTHTTPDoer
	baseURL    string
	apiKey     string
}

func NewDARTClient(client DARTHTTPDoer, baseURL, apiKey string) *DARTClient {
	return &DARTClient{httpClient: client, baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey}
}
func defaultDARTClient() *DARTClient {
	return NewDARTClient(httpclient.Default, "https://opendart.fss.or.kr", config.OpenDARTAPIKey)
}

type dartSearchInput struct {
	CompanyName string `json:"company_name" jsonschema:"기업명 또는 기업코드(8자리 숫자)"`
}

type dartDisclosuresInput struct {
	CompanyName string `json:"company_name" jsonschema:"기업명 또는 기업코드(8자리 숫자)"`
	StartDate   string `json:"start_date,omitempty" jsonschema:"조회 시작일 (YYYYMMDD, 예: 20240101)"`
	EndDate     string `json:"end_date,omitempty" jsonschema:"조회 종료일 (YYYYMMDD)"`
	NumOfRows   int    `json:"num_of_rows,omitempty" jsonschema:"결과 수 (기본: 10)"`
}

type dartFinancialsInput struct {
	CompanyName string `json:"company_name" jsonschema:"기업명 또는 기업코드(8자리 숫자)"`
	Year        string `json:"year" jsonschema:"사업연도 (YYYY, 예: 2023)"`
	ReportCode  string `json:"report_code,omitempty" jsonschema:"보고서 구분: 11011(사업보고서), 11012(반기보고서), 11013(1분기보고서), 11014(3분기보고서). 기본값: 11011"`
}

type dartDividendInput struct {
	CompanyName string `json:"company_name" jsonschema:"기업명 또는 기업코드(8자리 숫자)"`
	Year        string `json:"year" jsonschema:"사업연도 (YYYY, 예: 2023)"`
	ReportCode  string `json:"report_code,omitempty" jsonschema:"보고서 구분: 11011(사업보고서), 11012(반기보고서), 11013(1분기보고서), 11014(3분기보고서). 기본값: 11011"`
}

type dartMultiFinancialsInput struct {
	CompanyNames []string `json:"company_names" jsonschema:"비교할 기업명 또는 기업코드의 배열 (최대 10개)"`
	Year         string   `json:"year" jsonschema:"사업연도 (YYYY, 예: 2023)"`
	ReportCode   string   `json:"report_code,omitempty" jsonschema:"보고서 구분: 11011(사업보고서), 11012(반기보고서), 11013(1분기보고서), 11014(3분기보고서). 기본값: 11011"`
}

func RegisterDART(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_dart_company",
		Description: "DART에서 기업명으로 기업코드·기업개황(대표자, 주소, 업종 등) 조회.",
	}, withRecover("search_dart_company", searchDARTCompany))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_dart_disclosures",
		Description: "기업의 DART 공시 목록 조회 (사업보고서, 분기보고서 등).",
	}, withRecover("get_dart_disclosures", getDARTDisclosures))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_dart_financials",
		Description: "기업의 특정 연도/보고서별 재무제표(주요 계정) 조회.",
	}, withRecover("get_dart_financials", getDARTFinancials))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_dart_dividend_info",
		Description: "기업의 특정 연도/보고서별 배당에 관한 사항 조회.",
	}, withRecover("get_dart_dividend_info", getDARTDividendInfo))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_dart_multi_financials",
		Description: "여러 기업의 특정 연도/보고서별 주요 재무제표 계정 비교 조회.",
	}, withRecover("get_dart_multi_financials", getDARTMultiFinancials))
}

func dartCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".public-api-mcp", "dart_corp_codes.json")
}

func loadCorpCodes() (map[string]string, error) {
	dartCorpCodesMu.RLock()
	if dartCorpCodes != nil {
		dartCorpCodesMu.RUnlock()
		return dartCorpCodes, nil
	}
	dartCorpCodesMu.RUnlock()

	dartCorpCodesMu.Lock()
	defer dartCorpCodesMu.Unlock()

	cachePath := dartCachePath()
	info, err := os.Stat(cachePath)
	if err == nil && time.Since(info.ModTime()) < dartCacheTTL {
		data, err := os.ReadFile(cachePath)
		if err == nil {
			var codes map[string]string
			if json.Unmarshal(data, &codes) == nil {
				dartCorpCodes = codes
				slog.Info("DART 기업코드 파일캐시 로드", "count", len(codes), "path", cachePath)
				return codes, nil
			}
		}
	}

	if config.OpenDARTAPIKey == "" {
		return nil, fmt.Errorf("OPENDART_API_KEY가 설정되지 않았습니다")
	}

	slog.Info("DART 기업코드 다운로드 중")
	// 여러 요청이 공유하는 프로세스 전역 캐시를 채우는 백그라운드성 다운로드이므로
	// 특정 호출자의 컨텍스트가 아닌 자체 타임아웃으로 독립 실행한다 — 첫 호출자의
	// 요청이 취소돼도 캐시를 기다리는 다른 동시 호출자에 영향이 없어야 하기 때문.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", "https://opendart.fss.or.kr/api/corpCode.xml", nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}
	q := req.URL.Query()
	q.Set("crtfc_key", config.OpenDARTAPIKey)
	req.URL.RawQuery = q.Encode()

	zipData, _, err := httpclient.FetchBody(req, 50*1024*1024) // 50MB 제한
	if err != nil {
		return nil, fmt.Errorf("기업코드 다운로드 실패: %w", err)
	}

	codes, err := parseCorpCodeZip(zipData)
	if err != nil {
		return nil, err
	}

	os.MkdirAll(filepath.Dir(cachePath), 0755)
	data, _ := json.Marshal(codes)
	os.WriteFile(cachePath, data, 0644)

	dartCorpCodes = codes
	slog.Info("DART 기업코드 로드 완료", "count", len(codes))
	return codes, nil
}

func resolveCorpCode(nameOrCode string) (string, error) {
	s := strings.TrimSpace(nameOrCode)
	if len(s) == 8 {
		allDigits := true
		for _, c := range s {
			if c < '0' || c > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return s, nil
		}
	}

	if cached, ok := corpCodeCache.Get(s); ok {
		return cached, nil
	}

	codes, err := loadCorpCodes()
	if err != nil {
		return "", err
	}

	if code, ok := codes[s]; ok {
		corpCodeCache.Set(s, code, 24*time.Hour)
		return code, nil
	}

	// 부분 일치 검색
	var matches [][2]string
	for name, code := range codes {
		if strings.Contains(name, s) {
			matches = append(matches, [2]string{name, code})
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("'%s'에 해당하는 기업을 찾을 수 없습니다", nameOrCode)
	}
	// 가장 짧은(정확한) 이름 우선
	best := matches[0]
	for _, m := range matches[1:] {
		if len(m[0]) < len(best[0]) {
			best = m
		}
	}
	corpCodeCache.Set(s, best[1], 24*time.Hour)
	return best[1], nil
}

func dartGet(ctx context.Context, endpoint string, params map[string]string) (map[string]any, error) {
	return dartGetWithClient(ctx, defaultDARTClient(), endpoint, params)
}
func dartGetWithClient(ctx context.Context, client *DARTClient, endpoint string, params map[string]string) (map[string]any, error) {
	if client.apiKey == "" {
		return nil, fmt.Errorf("OPENDART_API_KEY가 설정되지 않았습니다")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", client.baseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}
	q := req.URL.Query()
	q.Set("crtfc_key", client.apiKey)
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
	if status, _ := data["status"].(string); status == "ERR" {
		msg, _ := data["message"].(string)
		return nil, fmt.Errorf("DART API 오류: %s", msg)
	}
	return data, nil
}

func searchDARTCompany(ctx context.Context, _ *mcp.CallToolRequest, in dartSearchInput) (*mcp.CallToolResult, any, error) {
	corpCode, err := resolveCorpCode(in.CompanyName)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	data, err := dartGet(ctx, "/api/company.json", map[string]string{"corp_code": corpCode})
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	var sb strings.Builder
	fields := []string{"corp_name", "corp_code", "stock_code", "ceo_nm", "corp_cls",
		"jurir_no", "bizr_no", "adres", "hm_url", "ir_url", "phn_no", "fax_no",
		"induty_code", "est_dt", "acc_mt"}
	labels := map[string]string{
		"corp_name": "기업명", "corp_code": "기업코드", "stock_code": "종목코드",
		"ceo_nm": "대표자", "corp_cls": "법인구분", "jurir_no": "법인등록번호",
		"bizr_no": "사업자등록번호", "adres": "주소", "hm_url": "홈페이지",
		"ir_url": "IR", "phn_no": "전화번호", "fax_no": "팩스",
		"induty_code": "업종코드", "est_dt": "설립일", "acc_mt": "결산월",
	}
	fmt.Fprintln(&sb, "### DART 기업개황")
	for _, f := range fields {
		if v, ok := data[f].(string); ok && v != "" {
			fmt.Fprintf(&sb, "- %s: %s\n", labels[f], v)
		}
	}
	return toolText(sb.String()), nil, nil
}

func getDARTDisclosures(ctx context.Context, _ *mcp.CallToolRequest, in dartDisclosuresInput) (*mcp.CallToolResult, any, error) {
	corpCode, err := resolveCorpCode(in.CompanyName)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	numOfRows := in.NumOfRows
	if numOfRows <= 0 {
		numOfRows = 10
	}
	params := map[string]string{
		"corp_code":  corpCode,
		"page_no":    "1",
		"page_count": fmt.Sprintf("%d", numOfRows),
	}
	if in.StartDate != "" {
		params["bgn_de"] = in.StartDate
	}
	if in.EndDate != "" {
		params["end_de"] = in.EndDate
	}

	data, err := dartGet(ctx, "/api/list.json", params)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	list, _ := data["list"].([]any)
	if len(list) == 0 {
		return toolText("공시 목록이 없습니다"), nil, nil
	}

	var sb strings.Builder
	fmt.Fprintln(&sb, "### DART 공시 목록")
	for _, raw := range list {
		item, _ := raw.(map[string]any)
		rcpDt, _ := item["rcept_dt"].(string)
		reportNm, _ := item["report_nm"].(string)
		flrNm, _ := item["flr_nm"].(string)
		rcpNo, _ := item["rcept_no"].(string)
		url := "https://dart.fss.or.kr/dsaf001/main.do?rcpNo=" + rcpNo
		fmt.Fprintf(&sb, "[%s] %s (제출: %s)\n%s\n\n", rcpDt, reportNm, flrNm, url)
	}
	return toolText(sb.String()), nil, nil
}

func getDARTFinancials(ctx context.Context, _ *mcp.CallToolRequest, in dartFinancialsInput) (*mcp.CallToolResult, any, error) {
	corpCode, err := resolveCorpCode(in.CompanyName)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	year := strings.TrimSpace(in.Year)
	if year == "" {
		return toolError("사업연도(year)는 필수 입력 항목입니다 (예: 2023)"), nil, nil
	}

	reportCode := in.ReportCode
	if reportCode == "" {
		reportCode = "11011"
	}

	params := map[string]string{
		"corp_code":  corpCode,
		"bsns_year":  year,
		"reprt_code": reportCode,
	}

	data, err := dartGet(ctx, "/api/fnlttSinglAcnt.json", params)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	list, _ := data["list"].([]any)
	if len(list) == 0 {
		return toolText("조회된 재무제표 데이터가 없습니다. 사업연도 또는 보고서 구분을 확인해 주세요."), nil, nil
	}

	var sb strings.Builder
	reportNames := map[string]string{
		"11011": "사업보고서",
		"11012": "반기보고서",
		"11013": "1분기보고서",
		"11014": "3분기보고서",
	}
	rName := reportNames[reportCode]
	if rName == "" {
		rName = reportCode
	}

	fmt.Fprintf(&sb, "### [%s] %s년 %s 주요 재무 정보\n\n", in.CompanyName, year, rName)

	bySj := make(map[string][]map[string]any)
	var sjOrder []string

	for _, raw := range list {
		item, _ := raw.(map[string]any)
		sjNm, _ := item["sj_nm"].(string)
		if sjNm == "" {
			sjNm = "기타"
		}
		if _, exists := bySj[sjNm]; !exists {
			sjOrder = append(sjOrder, sjNm)
		}
		bySj[sjNm] = append(bySj[sjNm], item)
	}

	for _, sjNm := range sjOrder {
		fmt.Fprintf(&sb, "#### %s\n", sjNm)
		fmt.Fprintln(&sb, "| 계정과목 | 당기 금액 | 전기 금액 | 전전기 금액 |")
		fmt.Fprintln(&sb, "|---|---|---|---|")
		for _, item := range bySj[sjNm] {
			accNm, _ := item["account_nm"].(string)
			thstrm, _ := item["thstrm_amount"].(string)
			frmtrm, _ := item["frmtrm_amount"].(string)
			bfefrmtrm, _ := item["bfefrmtrm_amount"].(string)

			if thstrm == "" {
				thstrm = "-"
			}
			if frmtrm == "" {
				frmtrm = "-"
			}
			if bfefrmtrm == "" {
				bfefrmtrm = "-"
			}

			fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n", accNm, thstrm, frmtrm, bfefrmtrm)
		}
		fmt.Fprintln(&sb)
	}

	return toolText(sb.String()), nil, nil
}

func getDARTDividendInfo(ctx context.Context, _ *mcp.CallToolRequest, in dartDividendInput) (*mcp.CallToolResult, any, error) {
	corpCode, err := resolveCorpCode(in.CompanyName)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	year := strings.TrimSpace(in.Year)
	if year == "" {
		return toolError("사업연도(year)는 필수 입력 항목입니다 (예: 2023)"), nil, nil
	}

	reportCode := in.ReportCode
	if reportCode == "" {
		reportCode = "11011"
	}

	params := map[string]string{
		"corp_code":  corpCode,
		"bsns_year":  year,
		"reprt_code": reportCode,
	}

	data, err := dartGet(ctx, "/api/alotMatter.json", params)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	list, _ := data["list"].([]any)
	if len(list) == 0 {
		return toolText("조회된 배당 정보 데이터가 없습니다. 사업연도 또는 보고서 구분을 확인해 주세요."), nil, nil
	}

	var sb strings.Builder
	reportNames := map[string]string{
		"11011": "사업보고서",
		"11012": "반기보고서",
		"11013": "1분기보고서",
		"11014": "3분기보고서",
	}
	rName := reportNames[reportCode]
	if rName == "" {
		rName = reportCode
	}

	fmt.Fprintf(&sb, "### [%s] %s년 %s 배당에 관한 사항\n\n", in.CompanyName, year, rName)
	fmt.Fprintln(&sb, "| 구분 | 당기 (thstrm) | 전기 (frmtrm) | 전전기 (bfefrmtrm) |")
	fmt.Fprintln(&sb, "|---|---|---|---|")

	for _, raw := range list {
		item, _ := raw.(map[string]any)
		se, _ := item["se"].(string)
		thstrm, _ := item["thstrm"].(string)
		frmtrm, _ := item["frmtrm"].(string)
		bfefrmtrm, _ := item["bfefrmtrm"].(string)

		if thstrm == "" {
			thstrm = "-"
		}
		if frmtrm == "" {
			frmtrm = "-"
		}
		if bfefrmtrm == "" {
			bfefrmtrm = "-"
		}

		fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n", se, thstrm, frmtrm, bfefrmtrm)
	}

	return toolText(sb.String()), nil, nil
}

func getDARTMultiFinancials(ctx context.Context, _ *mcp.CallToolRequest, in dartMultiFinancialsInput) (*mcp.CallToolResult, any, error) {
	if len(in.CompanyNames) == 0 {
		return toolError("비교할 기업명 목록(company_names)은 필수입니다"), nil, nil
	}
	if len(in.CompanyNames) > 10 {
		return toolError("한 번에 최대 10개 기업까지만 비교 가능합니다"), nil, nil
	}

	year := strings.TrimSpace(in.Year)
	if year == "" {
		return toolError("사업연도(year)는 필수 입력 항목입니다 (예: 2023)"), nil, nil
	}

	reportCode := in.ReportCode
	if reportCode == "" {
		reportCode = "11011"
	}

	var corpCodes []string
	for _, name := range in.CompanyNames {
		code, err := resolveCorpCode(name)
		if err != nil {
			return toolError("기업 '%s' 조회 실패: %v", name, err), nil, nil
		}
		corpCodes = append(corpCodes, code)
	}

	params := map[string]string{
		"corp_code":  strings.Join(corpCodes, ","),
		"bsns_year":  year,
		"reprt_code": reportCode,
	}

	data, err := dartGet(ctx, "/api/fnlttMultiAcnt.json", params)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	list, _ := data["list"].([]any)
	if len(list) == 0 {
		return toolText("조회된 재무제표 데이터가 없습니다. 사업연도 또는 보고서 구분을 확인해 주세요."), nil, nil
	}

	codes, err := loadCorpCodes()
	var nameMap map[string]string
	if err == nil {
		nameMap = make(map[string]string, len(codes))
		for name, code := range codes {
			nameMap[code] = name
		}
	}

	var sb strings.Builder
	reportNames := map[string]string{
		"11011": "사업보고서",
		"11012": "반기보고서",
		"11013": "1분기보고서",
		"11014": "3분기보고서",
	}
	rName := reportNames[reportCode]
	if rName == "" {
		rName = reportCode
	}

	fmt.Fprintf(&sb, "### %s년 %s 주요 재무 지표 비교\n\n", year, rName)
	fmt.Fprintln(&sb, "| 회사명 | 계정구분 | 계정과목 | 당기 금액 | 전기 금액 |")
	fmt.Fprintln(&sb, "|---|---|---|---|---|")

	for _, raw := range list {
		item, _ := raw.(map[string]any)
		cCode, _ := item["corp_code"].(string)
		cName := ""
		if nameMap != nil {
			cName = nameMap[cCode]
		}
		if cName == "" {
			cName = cCode
		}

		sjNm, _ := item["sj_nm"].(string)
		accNm, _ := item["account_nm"].(string)
		thstrm, _ := item["thstrm_amount"].(string)
		frmtrm, _ := item["frmtrm_amount"].(string)

		if thstrm == "" {
			thstrm = "-"
		}
		if frmtrm == "" {
			frmtrm = "-"
		}

		fmt.Fprintf(&sb, "| %s | %s | %s | %s | %s |\n", cName, sjNm, accNm, thstrm, frmtrm)
	}

	return toolText(sb.String()), nil, nil
}
