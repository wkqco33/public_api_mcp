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

type pixabayInput struct {
	Query     string `json:"query" jsonschema:"검색어 (영어로 입력 권장)"`
	ImageType string `json:"image_type,omitempty" jsonschema:"이미지 유형: all, photo, illustration, vector (기본: all)"`
	PerPage   int    `json:"per_page,omitempty" jsonschema:"결과 개수 3~200 (기본: 5)"`
}

func RegisterPixabay(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_pixabay_images",
		Description: "Pixabay에서 이미지를 검색합니다. 이미지 URL, 태그, 작성자 정보를 반환.",
	}, withRecover("search_pixabay_images", searchPixabayImages))
}

type PixabayHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}
type pixabayClient struct {
	httpClient PixabayHTTPDoer
	baseURL    string
	apiKey     string
}

func NewPixabayClient(client PixabayHTTPDoer, baseURL, apiKey string) *pixabayClient {
	return &pixabayClient{httpClient: client, baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey}
}
func defaultPixabayClient() *pixabayClient {
	return NewPixabayClient(httpclient.Default, "https://pixabay.com/api", config.PixabayAPIKey)
}

func searchPixabayImages(ctx context.Context, _ *mcp.CallToolRequest, in pixabayInput) (*mcp.CallToolResult, any, error) {
	return searchPixabayImagesWithClient(ctx, defaultPixabayClient(), in)
}
func searchPixabayImagesWithClient(ctx context.Context, client *pixabayClient, in pixabayInput) (*mcp.CallToolResult, any, error) {
	if client.apiKey == "" {
		return toolError("PIXABAY_API_KEY가 설정되지 않았습니다"), nil, nil
	}

	perPage := in.PerPage
	if perPage < 3 {
		perPage = 5
	}
	if perPage > 200 {
		perPage = 200
	}
	imageType := in.ImageType
	if imageType == "" {
		imageType = "all"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", client.baseURL+"/", nil)
	if err != nil {
		return toolError("요청 생성 실패: %v", err), nil, nil
	}
	q := req.URL.Query()
	q.Set("key", client.apiKey)
	q.Set("q", in.Query)
	q.Set("image_type", imageType)
	q.Set("per_page", fmt.Sprintf("%d", perPage))
	q.Set("safesearch", "true")
	req.URL.RawQuery = q.Encode()

	body, status, err := httpclient.FetchBodyWithClient(client.httpClient, req, maxAPIBodyBytes)
	if err != nil {
		return toolError("%v", err), nil, nil
	}
	if status != http.StatusOK {
		return toolError("Pixabay API 오류 [%d]: %s", status, string(body)), nil, nil
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return toolError("응답 파싱 실패: %v", err), nil, nil
	}

	hits, _ := data["hits"].([]any)
	if len(hits) == 0 {
		return toolText(fmt.Sprintf("'%s'에 대한 검색 결과가 없습니다", in.Query)), nil, nil
	}

	type imageResult struct {
		ID           float64 `json:"id"`
		PreviewURL   string  `json:"previewURL"`
		WebformatURL string  `json:"webformatURL"`
		LargeURL     string  `json:"largeImageURL"`
		Tags         string  `json:"tags"`
		User         string  `json:"user"`
	}

	var results []imageResult
	for _, raw := range hits {
		hit, _ := raw.(map[string]any)
		id, _ := hit["id"].(float64)
		results = append(results, imageResult{
			ID:           id,
			PreviewURL:   stringVal(hit, "previewURL"),
			WebformatURL: stringVal(hit, "webformatURL"),
			LargeURL:     stringVal(hit, "largeImageURL"),
			Tags:         stringVal(hit, "tags"),
			User:         stringVal(hit, "user"),
		})
	}

	out, _ := json.MarshalIndent(results, "", "  ")
	return toolText(string(out)), nil, nil
}

func stringVal(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	if v, ok := m[key].(float64); ok {
		return fmt.Sprintf("%.0f", v)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", m[key]))
}
