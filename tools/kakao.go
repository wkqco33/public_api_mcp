package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"wkqcosoft.com/public-api-mcp/config"
	"wkqcosoft.com/public-api-mcp/httpclient"
)

type kakaoSearchInput struct {
	Query string `json:"query" jsonschema:"검색어"`
	Size  int    `json:"size,omitempty" jsonschema:"결과 개수 (검색 종류별 최대값 적용, 기본: 10)"`
	Page  int    `json:"page,omitempty" jsonschema:"페이지 번호 (기본: 1)"`
	Sort  string `json:"sort,omitempty" jsonschema:"정렬: accuracy(정확도순), recency(최신순). 책은 latest(발간일순)"`
}

type kakaoPlaceInput struct {
	Query             string  `json:"query" jsonschema:"장소 검색어"`
	Longitude         float64 `json:"longitude,omitempty" jsonschema:"중심 경도 (선택)"`
	Latitude          float64 `json:"latitude,omitempty" jsonschema:"중심 위도 (선택)"`
	Radius            int     `json:"radius,omitempty" jsonschema:"검색 반경(미터, 0~20000)"`
	CategoryGroupCode string  `json:"category_group_code,omitempty" jsonschema:"카테고리 코드 (FD6 음식점, CE7 카페 등)"`
	Size              int     `json:"size,omitempty" jsonschema:"결과 개수 (최대 15)"`
	Page              int     `json:"page,omitempty" jsonschema:"페이지 번호 (1~45)"`
	Sort              string  `json:"sort,omitempty" jsonschema:"정렬: accuracy 또는 distance"`
}

type kakaoMeta struct {
	TotalCount    int  `json:"total_count"`
	PageableCount int  `json:"pageable_count"`
	IsEnd         bool `json:"is_end"`
}

type kakaoSearchResponse struct {
	Meta      kakaoMeta        `json:"meta"`
	Documents []map[string]any `json:"documents"`
}

func RegisterKakao(s *mcp.Server) {
	register := func(name, description, collection string) {
		mcp.AddTool(s, &mcp.Tool{Name: name, Description: description}, withRecover(name, func(ctx context.Context, req *mcp.CallToolRequest, in kakaoSearchInput) (*mcp.CallToolResult, any, error) {
			return kakaoSearch(ctx, collection, in)
		}))
	}
	register("kakao_search_web", "Daum 웹문서를 검색합니다.", "web")
	register("kakao_search_blog", "Daum 블로그 게시글을 검색합니다.", "blog")
	register("kakao_search_cafe", "Daum 카페 게시글을 검색합니다.", "cafe")
	register("kakao_search_book", "Daum 도서 정보를 검색합니다.", "book")
	register("kakao_search_image", "Daum 이미지를 검색합니다.", "image")
	register("kakao_search_video", "동영상 검색 결과를 조회합니다.", "vclip")
	mcp.AddTool(s, &mcp.Tool{Name: "kakao_search_places", Description: "카카오맵 장소를 키워드로 검색합니다. 좌표와 반경을 지정하면 주변 장소를 검색합니다."}, withRecover("kakao_search_places", kakaoSearchPlaces))
}

func kakaoSearch(ctx context.Context, collection string, in kakaoSearchInput) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(in.Query) == "" {
		return toolError("query는 비어 있을 수 없습니다"), nil, nil
	}
	page, size, sort, err := normalizeKakaoSearch(collection, in.Page, in.Size, in.Sort)
	if err != nil {
		return toolError("잘못된 검색 옵션: %v", err), nil, nil
	}
	return kakaoSearchRequest(ctx, collection, in.Query, page, size, sort)
}

func normalizeKakaoSearch(collection string, page, size int, sort string) (int, int, string, error) {
	if page <= 0 {
		page = 1
	}
	maxPage := 50
	if collection == "vclip" {
		maxPage = 15
	}
	if page > maxPage {
		return 0, 0, "", fmt.Errorf("page는 %d 이하이어야 합니다", maxPage)
	}
	if size <= 0 {
		size = 10
	}
	maxSize := 50
	if collection == "image" {
		maxSize = 80
	}
	if collection == "vclip" {
		maxSize = 30
	}
	if size > maxSize {
		return 0, 0, "", fmt.Errorf("size는 %d 이하이어야 합니다", maxSize)
	}
	if sort == "" {
		sort = "accuracy"
	}
	if collection == "book" {
		if sort != "accuracy" && sort != "latest" {
			return 0, 0, "", fmt.Errorf("책 sort는 accuracy 또는 latest이어야 합니다")
		}
	} else if sort != "accuracy" && sort != "recency" {
		return 0, 0, "", fmt.Errorf("sort는 accuracy 또는 recency이어야 합니다")
	}
	return page, size, sort, nil
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type kakaoClient struct {
	httpClient httpDoer
	baseURL    string
	apiKey     string
}

func newKakaoClient(client httpDoer, baseURL, apiKey string) *kakaoClient {
	return &kakaoClient{httpClient: client, baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey}
}

func defaultKakaoClient() *kakaoClient {
	return newKakaoClient(httpclient.Default, "https://dapi.kakao.com", config.KakaoRESTAPIKey)
}

func (c *kakaoClient) do(req *http.Request) ([]byte, int, error) {
	if c.apiKey == "" {
		return nil, 0, fmt.Errorf("KAKAO_REST_API_KEY가 설정되지 않았습니다")
	}
	req.Header.Set("Authorization", "KakaoAK "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	return httpclient.FetchBodyWithClient(c.httpClient, req, maxAPIBodyBytes)
}

func kakaoSearchRequest(ctx context.Context, collection, query string, page, size int, sort string) (*mcp.CallToolResult, any, error) {
	return kakaoSearchRequestWithClient(ctx, defaultKakaoClient(), collection, query, page, size, sort)
}

func kakaoSearchRequestWithClient(ctx context.Context, client *kakaoClient, collection, query string, page, size int, sort string) (*mcp.CallToolResult, any, error) {
	version := "v2"
	if collection == "book" {
		version = "v3"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/"+version+"/search/"+collection, nil)
	if err != nil {
		return toolError("요청 생성 실패: %v", err), nil, nil
	}
	q := req.URL.Query()
	q.Set("query", query)
	q.Set("page", fmt.Sprint(page))
	q.Set("size", fmt.Sprint(size))
	q.Set("sort", sort)
	req.URL.RawQuery = q.Encode()
	body, status, err := client.do(req)
	if err != nil {
		return toolError("카카오 API 요청 실패: %v", err), nil, nil
	}
	if status != http.StatusOK {
		return toolError("카카오 API 오류 [%d]: %s", status, string(body)), nil, nil
	}
	var data kakaoSearchResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return toolError("카카오 응답 파싱 실패: %v", err), nil, nil
	}
	if len(data.Documents) == 0 {
		return toolText(fmt.Sprintf("'%s'에 대한 검색 결과가 없습니다.", query)), nil, nil
	}
	return toolText(formatKakaoDocuments(collection, data.Documents)), nil, nil
}

func kakaoSearchPlaces(ctx context.Context, req *mcp.CallToolRequest, in kakaoPlaceInput) (*mcp.CallToolResult, any, error) {
	return kakaoSearchPlacesWithClient(ctx, defaultKakaoClient(), req, in)
}

func kakaoSearchPlacesWithClient(ctx context.Context, client *kakaoClient, _ *mcp.CallToolRequest, in kakaoPlaceInput) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(in.Query) == "" {
		return toolError("query는 비어 있을 수 없습니다"), nil, nil
	}
	if in.Size <= 0 {
		in.Size = 15
	}
	if in.Size > 15 || in.Page < 0 || in.Page > 45 {
		return toolError("size는 15 이하, page는 1~45 범위이어야 합니다"), nil, nil
	}
	if in.Page == 0 {
		in.Page = 1
	}
	if in.Radius < 0 || in.Radius > 20000 {
		return toolError("radius는 0~20000 범위이어야 합니다"), nil, nil
	}
	hasCoords := in.Longitude != 0 || in.Latitude != 0
	if hasCoords && (in.Longitude < -180 || in.Longitude > 180 || in.Latitude < -90 || in.Latitude > 90) {
		return toolError("좌표 범위가 올바르지 않습니다"), nil, nil
	}
	if in.Radius > 0 && !hasCoords {
		return toolError("radius를 사용하려면 longitude와 latitude가 필요합니다"), nil, nil
	}
	path := client.baseURL + "/v2/local/search/keyword.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return toolError("요청 생성 실패: %v", err), nil, nil
	}
	q := req.URL.Query()
	q.Set("query", in.Query)
	q.Set("page", fmt.Sprint(in.Page))
	q.Set("size", fmt.Sprint(in.Size))
	if in.CategoryGroupCode != "" {
		q.Set("category_group_code", in.CategoryGroupCode)
	}
	if hasCoords {
		q.Set("x", fmt.Sprint(in.Longitude))
		q.Set("y", fmt.Sprint(in.Latitude))
	}
	if in.Radius > 0 {
		q.Set("radius", fmt.Sprint(in.Radius))
	}
	if in.Sort != "" {
		if in.Sort != "accuracy" && in.Sort != "distance" {
			return toolError("sort는 accuracy 또는 distance이어야 합니다"), nil, nil
		}
		q.Set("sort", in.Sort)
	}
	req.URL.RawQuery = q.Encode()
	body, status, err := client.do(req)
	if err != nil {
		return toolError("카카오 API 요청 실패: %v", err), nil, nil
	}
	if status != http.StatusOK {
		return toolError("카카오 API 오류 [%d]: %s", status, string(body)), nil, nil
	}
	var data kakaoSearchResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return toolError("카카오 응답 파싱 실패: %v", err), nil, nil
	}
	if len(data.Documents) == 0 {
		return toolText(fmt.Sprintf("'%s'에 대한 장소 검색 결과가 없습니다.", in.Query)), nil, nil
	}
	return toolText(formatKakaoPlaces(data.Documents)), nil, nil
}

func formatKakaoDocuments(collection string, docs []map[string]any) string {
	var sb strings.Builder
	for _, d := range docs {
		title := stripHTMLTags(kakaoString(d, "title"))
		if title == "" {
			title = kakaoString(d, "name")
		}
		link := kakaoString(d, "url")
		if link == "" {
			link = kakaoString(d, "doc_url")
		}
		contents := stripHTMLTags(kakaoString(d, "contents"))
		fmt.Fprintf(&sb, "제목: %s\n링크: %s\n설명: %s", title, link, contents)
		if collection == "blog" {
			fmt.Fprintf(&sb, "\n블로그: %s", kakaoString(d, "blogname"))
		}
		if collection == "cafe" {
			fmt.Fprintf(&sb, "\n카페: %s", kakaoString(d, "cafename"))
		}
		if collection == "book" {
			fmt.Fprintf(&sb, "\n저자: %s\n출판사: %s\nISBN: %s", kakaoString(d, "authors"), kakaoString(d, "publisher"), kakaoString(d, "isbn"))
		}
		if collection == "image" {
			fmt.Fprintf(&sb, "\n이미지: %s\n출처: %s", kakaoString(d, "image_url"), kakaoString(d, "display_sitename"))
		}
		if collection == "vclip" {
			fmt.Fprintf(&sb, "\n업로더: %s\n썸네일: %s", kakaoString(d, "author"), kakaoString(d, "thumbnail"))
		}
		sb.WriteString("\n---\n")
	}
	return sb.String()
}

func formatKakaoPlaces(docs []map[string]any) string {
	var sb strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&sb, "장소: %s\n카테고리: %s\n전화: %s\n주소: %s\n도로명: %s\n좌표: %s, %s\n링크: %s\n---\n", kakaoString(d, "place_name"), kakaoString(d, "category_name"), kakaoString(d, "phone"), kakaoString(d, "address_name"), kakaoString(d, "road_address_name"), kakaoString(d, "x"), kakaoString(d, "y"), kakaoString(d, "place_url"))
	}
	return sb.String()
}

func stripHTMLTags(s string) string {
	s = strings.ReplaceAll(s, "<b>", "")
	s = strings.ReplaceAll(s, "</b>", "")
	return html.UnescapeString(s)
}

func kakaoString(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return html.UnescapeString(s)
	}
	if a, ok := m[key].([]any); ok {
		var values []string
		for _, v := range a {
			if s, ok := v.(string); ok {
				values = append(values, s)
			}
		}
		return strings.Join(values, ", ")
	}
	return ""
}
