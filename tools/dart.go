package tools

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"wkqcosoft.com/public-api-mcp/config"
	"wkqcosoft.com/public-api-mcp/httpclient"
)

const dartCacheTTL = 7 * 24 * time.Hour

var (
	dartCorpCodes   map[string]string
	dartCorpCodesMu sync.RWMutex
)

type dartSearchInput struct {
	CompanyName string `json:"company_name" jsonschema:"기업명 또는 기업코드(8자리 숫자)"`
}

type dartDisclosuresInput struct {
	CompanyName string `json:"company_name" jsonschema:"기업명 또는 기업코드(8자리 숫자)"`
	StartDate   string `json:"start_date,omitempty" jsonschema:"조회 시작일 (YYYYMMDD, 예: 20240101)"`
	EndDate     string `json:"end_date,omitempty" jsonschema:"조회 종료일 (YYYYMMDD)"`
	NumOfRows   int    `json:"num_of_rows,omitempty" jsonschema:"결과 수 (기본: 10)"`
}

func RegisterDART(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_dart_company",
		Description: "DART에서 기업명으로 기업코드·기업개황(대표자, 주소, 업종 등) 조회.",
	}, searchDARTCompany)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_dart_disclosures",
		Description: "기업의 DART 공시 목록 조회 (사업보고서, 분기보고서 등).",
	}, getDARTDisclosures)
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
	req, _ := http.NewRequest("GET", "https://opendart.fss.or.kr/api/corpCode.xml", nil)
	q := req.URL.Query()
	q.Set("crtfc_key", config.OpenDARTAPIKey)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("기업코드 다운로드 실패: %v", err)
	}
	defer res.Body.Close()
	zipData, _ := io.ReadAll(res.Body)

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("ZIP 파싱 실패: %v", err)
	}

	var xmlBytes []byte
	for _, f := range zr.File {
		if strings.HasSuffix(strings.ToUpper(f.Name), ".XML") {
			rc, _ := f.Open()
			xmlBytes, _ = io.ReadAll(rc)
			rc.Close()
			break
		}
	}

	type corpList struct {
		Items []struct {
			Name string `xml:"corp_name"`
			Code string `xml:"corp_code"`
		} `xml:"list"`
	}
	var cl corpList
	if err := xml.Unmarshal(xmlBytes, &cl); err != nil {
		return nil, fmt.Errorf("XML 파싱 실패: %v", err)
	}

	codes := make(map[string]string, len(cl.Items))
	for _, item := range cl.Items {
		name := strings.TrimSpace(item.Name)
		code := strings.TrimSpace(item.Code)
		if name != "" && code != "" {
			codes[name] = code
		}
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

	codes, err := loadCorpCodes()
	if err != nil {
		return "", err
	}

	if code, ok := codes[s]; ok {
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
	return best[1], nil
}

func dartGet(endpoint string, params map[string]string) (map[string]any, error) {
	if config.OpenDARTAPIKey == "" {
		return nil, fmt.Errorf("OPENDART_API_KEY가 설정되지 않았습니다")
	}
	req, _ := http.NewRequest("GET", "https://opendart.fss.or.kr"+endpoint, nil)
	q := req.URL.Query()
	q.Set("crtfc_key", config.OpenDARTAPIKey)
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	res, err := httpclient.Default.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 요청 실패: %v", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

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

func searchDARTCompany(_ context.Context, _ *mcp.CallToolRequest, in dartSearchInput) (*mcp.CallToolResult, any, error) {
	corpCode, err := resolveCorpCode(in.CompanyName)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	data, err := dartGet("/api/company.json", map[string]string{"corp_code": corpCode})
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

func getDARTDisclosures(_ context.Context, _ *mcp.CallToolRequest, in dartDisclosuresInput) (*mcp.CallToolResult, any, error) {
	corpCode, err := resolveCorpCode(in.CompanyName)
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	numOfRows := in.NumOfRows
	if numOfRows <= 0 {
		numOfRows = 10
	}
	params := map[string]string{
		"corp_code": corpCode,
		"page_no":   "1",
		"page_count": fmt.Sprintf("%d", numOfRows),
	}
	if in.StartDate != "" {
		params["bgn_de"] = in.StartDate
	}
	if in.EndDate != "" {
		params["end_de"] = in.EndDate
	}

	data, err := dartGet("/api/list.json", params)
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
