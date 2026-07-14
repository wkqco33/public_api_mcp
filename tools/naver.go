package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"wkqcosoft.com/public-api-mcp/config"
	"wkqcosoft.com/public-api-mcp/httpclient"
)

type naverSearchInput struct {
	Query   string `json:"query" jsonschema:"검색어"`
	Display int    `json:"display,omitempty" jsonschema:"결과 개수 (기본: 10, 최대: 100)"`
	Start   int    `json:"start,omitempty" jsonschema:"시작 위치 (기본: 1)"`
	Sort    string `json:"sort,omitempty" jsonschema:"정렬 방식: sim(정확도순), date(날짜순)"`
}

type naverQueryOnlyInput struct {
	Query string `json:"query" jsonschema:"검색어"`
}

func RegisterNaver(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{Name: "naver_search_blog", Description: "네이버 블로그 게시글 검색. 상품 후기, 맛집 경험담, 여행기 등 사용자 의견 수집에 사용."}, withRecover("naver_search_blog", naverSearchBlog))
	mcp.AddTool(s, &mcp.Tool{Name: "naver_search_news", Description: "한국 최신 뉴스·기사 검색. 실시간 사건·사고, 시사 이슈, 특정 인물·기업 뉴스 검색에 사용."}, withRecover("naver_search_news", naverSearchNews))
	mcp.AddTool(s, &mcp.Tool{Name: "naver_search_cafe", Description: "네이버 카페 게시글 검색. 커뮤니티 의견, 맘카페 등 실사용자 의견 수집에 사용."}, withRecover("naver_search_cafe", naverSearchCafe))
	mcp.AddTool(s, &mcp.Tool{Name: "naver_search_book", Description: "네이버 책 정보 검색(가격, ISBN, 출판사, 저자). 특정 도서 정보 조회에 사용."}, withRecover("naver_search_book", naverSearchBook))
	mcp.AddTool(s, &mcp.Tool{Name: "naver_search_encyc", Description: "네이버 백과사전 검색. 용어 설명, 역사 인물, 지식 정의 검색에 사용."}, withRecover("naver_search_encyc", naverSearchEncyc))
	mcp.AddTool(s, &mcp.Tool{Name: "naver_search_kin", Description: "네이버 지식iN 질문·답변 검색. 실용적인 질문 해결에 활용."}, withRecover("naver_search_kin", naverSearchKin))
	mcp.AddTool(s, &mcp.Tool{Name: "naver_search_local", Description: "네이버 지역·장소 검색. 맛집, 카페, 병원 등 주변 업체 검색에 사용."}, withRecover("naver_search_local", naverSearchLocal))
	mcp.AddTool(s, &mcp.Tool{Name: "naver_search_webkr", Description: "네이버 웹문서 전반 검색. 일반 한국어 웹 검색에 사용."}, withRecover("naver_search_webkr", naverSearchWebkr))
	mcp.AddTool(s, &mcp.Tool{Name: "naver_search_image", Description: "네이버 이미지 검색. 한국 관련 이미지, 인물, 상품 사진 검색."}, withRecover("naver_search_image", naverSearchImage))
	mcp.AddTool(s, &mcp.Tool{Name: "naver_search_shop", Description: "네이버 쇼핑 최저가·판매처 검색. 상품 구매, 가격 비교에 사용."}, withRecover("naver_search_shop", naverSearchShop))
	mcp.AddTool(s, &mcp.Tool{Name: "naver_search_doc", Description: "네이버 전문자료(논문, 학술자료) 검색. 학술·연구 자료 수집에 사용."}, withRecover("naver_search_doc", naverSearchDoc))
	mcp.AddTool(s, &mcp.Tool{Name: "naver_check_adult", Description: "입력한 검색어가 성인 검색어인지 판별."}, withRecover("naver_check_adult", naverCheckAdult))
	mcp.AddTool(s, &mcp.Tool{Name: "naver_convert_errata", Description: "검색어의 오타를 교정하여 제안어를 반환."}, withRecover("naver_convert_errata", naverConvertErrata))
}

func naverSearchBlog(ctx context.Context, _ *mcp.CallToolRequest, in naverSearchInput) (*mcp.CallToolResult, any, error) {
	return naverSearchResult(ctx, "blog", in.Query, in.Display, in.Start, in.Sort)
}

func naverSearchNews(ctx context.Context, _ *mcp.CallToolRequest, in naverSearchInput) (*mcp.CallToolResult, any, error) {
	return naverSearchResult(ctx, "news", in.Query, in.Display, in.Start, in.Sort)
}

func naverSearchCafe(ctx context.Context, _ *mcp.CallToolRequest, in naverSearchInput) (*mcp.CallToolResult, any, error) {
	return naverSearchResult(ctx, "cafearticle", in.Query, in.Display, in.Start, in.Sort)
}

func naverSearchBook(ctx context.Context, _ *mcp.CallToolRequest, in naverSearchInput) (*mcp.CallToolResult, any, error) {
	return naverSearchResult(ctx, "book", in.Query, in.Display, in.Start, in.Sort)
}

func naverSearchEncyc(ctx context.Context, _ *mcp.CallToolRequest, in naverSearchInput) (*mcp.CallToolResult, any, error) {
	return naverSearchResult(ctx, "encyc", in.Query, in.Display, in.Start, "")
}

func naverSearchKin(ctx context.Context, _ *mcp.CallToolRequest, in naverSearchInput) (*mcp.CallToolResult, any, error) {
	return naverSearchResult(ctx, "kin", in.Query, in.Display, in.Start, in.Sort)
}

func naverSearchLocal(ctx context.Context, _ *mcp.CallToolRequest, in naverSearchInput) (*mcp.CallToolResult, any, error) {
	sort := in.Sort
	if sort == "" {
		sort = "random"
	}
	return naverSearchResult(ctx, "local", in.Query, in.Display, in.Start, sort)
}

func naverSearchWebkr(ctx context.Context, _ *mcp.CallToolRequest, in naverSearchInput) (*mcp.CallToolResult, any, error) {
	return naverSearchResult(ctx, "webkr", in.Query, in.Display, in.Start, "")
}

func naverSearchImage(ctx context.Context, _ *mcp.CallToolRequest, in naverSearchInput) (*mcp.CallToolResult, any, error) {
	return naverSearchResult(ctx, "image", in.Query, in.Display, in.Start, in.Sort)
}

func naverSearchShop(ctx context.Context, _ *mcp.CallToolRequest, in naverSearchInput) (*mcp.CallToolResult, any, error) {
	return naverSearchResult(ctx, "shop", in.Query, in.Display, in.Start, in.Sort)
}

func naverSearchDoc(ctx context.Context, _ *mcp.CallToolRequest, in naverSearchInput) (*mcp.CallToolResult, any, error) {
	return naverSearchResult(ctx, "doc", in.Query, in.Display, in.Start, "")
}

func naverCheckAdult(ctx context.Context, _ *mcp.CallToolRequest, in naverQueryOnlyInput) (*mcp.CallToolResult, any, error) {
	if config.NaverClientID == "" || config.NaverClientSecret == "" {
		return toolError("NAVER_CLIENT_ID 또는 NAVER_CLIENT_SECRET이 설정되지 않았습니다"), nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://openapi.naver.com/v1/util/adult.json", nil)
	if err != nil {
		return toolError("요청 생성 실패: %v", err), nil, nil
	}
	req.Header.Set("X-Naver-Client-Id", config.NaverClientID)
	req.Header.Set("X-Naver-Client-Secret", config.NaverClientSecret)
	q := req.URL.Query()
	q.Set("query", in.Query)
	req.URL.RawQuery = q.Encode()

	body, _, err := httpclient.FetchBody(req, maxAPIBodyBytes)
	if err != nil {
		return toolError("%v", err), nil, nil
	}
	var data map[string]string
	if err := json.Unmarshal(body, &data); err != nil {
		return toolError("응답 파싱 실패: %v", err), nil, nil
	}
	if data["adult"] == "1" {
		return toolText("성인 검색어입니다."), nil, nil
	}
	return toolText("일반 검색어입니다."), nil, nil
}

func naverConvertErrata(ctx context.Context, _ *mcp.CallToolRequest, in naverQueryOnlyInput) (*mcp.CallToolResult, any, error) {
	if config.NaverClientID == "" || config.NaverClientSecret == "" {
		return toolError("NAVER_CLIENT_ID 또는 NAVER_CLIENT_SECRET이 설정되지 않았습니다"), nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://openapi.naver.com/v1/util/errata.json", nil)
	if err != nil {
		return toolError("요청 생성 실패: %v", err), nil, nil
	}
	req.Header.Set("X-Naver-Client-Id", config.NaverClientID)
	req.Header.Set("X-Naver-Client-Secret", config.NaverClientSecret)
	q := req.URL.Query()
	q.Set("query", in.Query)
	req.URL.RawQuery = q.Encode()

	body, _, err := httpclient.FetchBody(req, maxAPIBodyBytes)
	if err != nil {
		return toolError("%v", err), nil, nil
	}
	var data map[string]string
	if err := json.Unmarshal(body, &data); err != nil {
		return toolError("응답 파싱 실패: %v", err), nil, nil
	}
	if data["errata"] == "" {
		return toolText("교정 제안이 없습니다."), nil, nil
	}
	return toolText("교정 제안: " + data["errata"]), nil, nil
}

func naverSearchResult(ctx context.Context, searchType, query string, display, start int, sort string) (*mcp.CallToolResult, any, error) {
	if config.NaverClientID == "" || config.NaverClientSecret == "" {
		return toolError("NAVER_CLIENT_ID 또는 NAVER_CLIENT_SECRET이 설정되지 않았습니다"), nil, nil
	}
	if display <= 0 {
		display = 10
	}
	if start <= 0 {
		start = 1
	}

	apiURL := fmt.Sprintf("https://openapi.naver.com/v1/search/%s.json", searchType)
	params := url.Values{}
	params.Set("query", query)
	params.Set("display", fmt.Sprintf("%d", display))
	params.Set("start", fmt.Sprintf("%d", start))
	if sort != "" {
		params.Set("sort", sort)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return toolError("요청 생성 실패: %v", err), nil, nil
	}
	req.Header.Set("X-Naver-Client-Id", config.NaverClientID)
	req.Header.Set("X-Naver-Client-Secret", config.NaverClientSecret)

	body, status, err := httpclient.FetchBody(req, maxAPIBodyBytes)
	if err != nil {
		return toolError("%v", err), nil, nil
	}
	if status != http.StatusOK {
		return toolError("네이버 API 오류 [%d]: %s", status, string(body)), nil, nil
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return toolError("응답 파싱 실패: %v", err), nil, nil
	}

	items, _ := data["items"].([]any)
	if len(items) == 0 {
		return toolText(fmt.Sprintf("'%s'에 대한 검색 결과가 없습니다.", query)), nil, nil
	}

	var sb strings.Builder
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		title := stripHTMLTags(strVal(item, "title"))
		link := strVal(item, "link")
		desc := stripHTMLTags(strVal(item, "description"))

		fmt.Fprintf(&sb, "제목: %s\n링크: %s\n설명: %s", title, link, desc)

		switch searchType {
		case "blog":
			fmt.Fprintf(&sb, "\n블로거: %s\n날짜: %s", strVal(item, "bloggername"), strVal(item, "postdate"))
		case "news":
			fmt.Fprintf(&sb, "\n언론사: %s\n날짜: %s", strVal(item, "originallink"), strVal(item, "pubDate"))
		case "cafearticle":
			fmt.Fprintf(&sb, "\n카페명: %s\n작성자: %s", strVal(item, "cafename"), strVal(item, "author"))
		case "book":
			fmt.Fprintf(&sb, "\n저자: %s\n출판사: %s\n가격: %s원", strVal(item, "author"), strVal(item, "publisher"), strVal(item, "discount"))
		case "local":
			fmt.Fprintf(&sb, "\n주소: %s\n도로명: %s\n카테고리: %s", strVal(item, "address"), strVal(item, "roadAddress"), strVal(item, "category"))
		case "shop":
			fmt.Fprintf(&sb, "\n최저가: %s원\n몰이름: %s\n카테고리: %s>%s", strVal(item, "lprice"), strVal(item, "mallName"), strVal(item, "category1"), strVal(item, "category2"))
		}
		sb.WriteString("\n---\n")
	}

	return toolText(sb.String()), nil, nil
}

func strVal(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func stripHTMLTags(s string) string {
	s = strings.ReplaceAll(s, "<b>", "")
	s = strings.ReplaceAll(s, "</b>", "")
	return html.UnescapeString(s)
}
