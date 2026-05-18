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

var bookCategoryMap = map[string]string{
	"문학":    "11",
	"인문과학":  "6",
	"사회과학":  "5",
	"자연과학":  "4",
}

type bookInput struct {
	Category  string `json:"category,omitempty" jsonschema:"도서 분류: 문학, 인문과학, 사회과학, 자연과학"`
	StartDate string `json:"start_date,omitempty" jsonschema:"조회 시작일 (YYYYMMDD)"`
	EndDate   string `json:"end_date,omitempty" jsonschema:"조회 종료일 (YYYYMMDD)"`
	StartNum  int    `json:"start_num,omitempty" jsonschema:"시작 번호 (기본: 1)"`
	EndNum    int    `json:"end_num,omitempty" jsonschema:"종료 번호 (기본: 5)"`
}

type nlXMLResponse struct {
	TotalCount string    `xml:"totalCount"`
	Items      []nlXMLItem `xml:"item"`
}

type nlXMLItem struct {
	Title      string `xml:"recomtitle"`
	Author     string `xml:"recomauthor"`
	Publisher  string `xml:"recompublisher"`
	Year       string `xml:"publishYear"`
	CategoryNm string `xml:"drCodeName"`
	ISBN       string `xml:"recom_isbn"`
	Contents   string `xml:"recomcontens"`
}

func RegisterBook(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_recommended_books",
		Description: "국립중앙도서관 사서 추천 도서 목록 조회. 분류별 추천 도서를 확인할 때 사용.",
	}, getRecommendedBooks)
}

func getRecommendedBooks(_ context.Context, _ *mcp.CallToolRequest, in bookInput) (*mcp.CallToolResult, any, error) {
	if config.NationalLibraryAPIKey == "" {
		return toolError("NATIONAL_LIBRARY_API_KEY가 설정되지 않았습니다"), nil, nil
	}

	startNum := in.StartNum
	if startNum <= 0 {
		startNum = 1
	}
	endNum := in.EndNum
	if endNum <= 0 {
		endNum = 5
	}

	req, _ := http.NewRequest("GET", "https://nl.go.kr/NL/search/openApi/saseoApi.do", nil)
	q := req.URL.Query()
	q.Set("Key", config.NationalLibraryAPIKey)
	q.Set("startRowNumApi", fmt.Sprintf("%d", startNum))
	q.Set("endRowNumApi", fmt.Sprintf("%d", endNum))
	if code, ok := bookCategoryMap[in.Category]; ok {
		q.Set("drCode", code)
	}
	if in.StartDate != "" {
		q.Set("start_date", in.StartDate)
	}
	if in.EndDate != "" {
		q.Set("end_date", in.EndDate)
	}
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

	var resp nlXMLResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return toolError("XML 파싱 실패: %v", err), nil, nil
	}
	if len(resp.Items) == 0 {
		return toolText("추천 도서 결과가 없습니다"), nil, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "### 국립중앙도서관 사서추천도서 (전체 %s건 중 %d건) ###\n\n", resp.TotalCount, len(resp.Items))
	for i, item := range resp.Items {
		contents := strings.TrimSpace(item.Contents)
		// HTML 태그 제거
		contents = extractText(contents)
		if len(contents) > 200 {
			contents = contents[:200] + "..."
		}
		fmt.Fprintf(&sb, "%d. %s\n   - 저자: %s | 출판사: %s (%s)\n   - 분류: %s | ISBN: %s\n   - 내용: %s\n\n",
			i+1, item.Title, item.Author, item.Publisher, item.Year, item.CategoryNm, item.ISBN, contents)
	}
	return toolText(sb.String()), nil, nil
}
