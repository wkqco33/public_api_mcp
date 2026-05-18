package tools

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"wkqcosoft.com/public-api-mcp/cache"
	"wkqcosoft.com/public-api-mcp/httpclient"
)

var newsCache = cache.New[string, string]()

var (
	reScript = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reNav    = regexp.MustCompile(`(?is)<nav[^>]*>.*?</nav>`)
	reFooter = regexp.MustCompile(`(?is)<footer[^>]*>.*?</footer>`)
	reHeader = regexp.MustCompile(`(?is)<header[^>]*>.*?</header>`)
	reTag    = regexp.MustCompile(`<[^>]*>`)
)

type getNewsInput struct {
	Topic string `json:"topic" jsonschema:"뉴스 주제 또는 키워드"`
}

type fetchURLInput struct {
	URL string `json:"url" jsonschema:"가져올 웹페이지 URL"`
}

func RegisterWeb(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{Name: "get_news", Description: "특정 주제의 최신 뉴스를 Google News RSS에서 가져옵니다."}, getNews)
	mcp.AddTool(s, &mcp.Tool{Name: "fetch_url", Description: "URL의 웹페이지 본문 텍스트를 추출합니다. 기사, 문서, 웹페이지 내용 확인에 사용."}, fetchURL)
}

type rssItem struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
}

type rssFeed struct {
	Items []rssItem `xml:"channel>item"`
}

func getNews(_ context.Context, _ *mcp.CallToolRequest, in getNewsInput) (*mcp.CallToolResult, any, error) {
	if cached, ok := newsCache.Get(in.Topic); ok {
		slog.Debug("뉴스 캐시 hit", "topic", in.Topic)
		return toolText(cached), nil, nil
	}
	slog.Debug("뉴스 RSS 조회", "topic", in.Topic)

	apiURL := fmt.Sprintf("https://news.google.com/rss/search?q=%s&hl=ko&gl=KR&ceid=KR:ko",
		url.QueryEscape(in.Topic))

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", httpclient.UserAgent)

	res, err := httpclient.Default.Do(req)
	if err != nil {
		return toolError("뉴스 요청 실패: %v", err), nil, nil
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return toolError("RSS 파싱 실패: %v", err), nil, nil
	}
	if len(feed.Items) == 0 {
		return toolText("뉴스 없음"), nil, nil
	}

	var sb strings.Builder
	limit := 5
	if len(feed.Items) < limit {
		limit = len(feed.Items)
	}
	for _, item := range feed.Items[:limit] {
		fmt.Fprintf(&sb, "- %s\n  URL: %s\n", item.Title, item.Link)
	}

	result := sb.String()
	newsCache.Set(in.Topic, result, 10*time.Minute)
	return toolText(result), nil, nil
}

func fetchURL(_ context.Context, _ *mcp.CallToolRequest, in fetchURLInput) (*mcp.CallToolResult, any, error) {
	req, err := http.NewRequest("GET", in.URL, nil)
	if err != nil {
		return toolError("잘못된 URL: %v", err), nil, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	req = req.WithContext(ctx)
	res, err := httpclient.Default.Do(req)
	if err != nil {
		return toolError("URL 요청 실패: %v", err), nil, nil
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return toolError("HTTP %d 오류", res.StatusCode), nil, nil
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20)) // 1MB 제한
	if err != nil {
		return toolError("본문 읽기 실패: %v", err), nil, nil
	}

	text := extractText(string(body))
	if len(text) > 10000 {
		text = text[:10000]
	}
	return toolText(text), nil, nil
}

func extractText(html string) string {
	html = reScript.ReplaceAllString(html, " ")
	html = reStyle.ReplaceAllString(html, " ")
	html = reNav.ReplaceAllString(html, " ")
	html = reFooter.ReplaceAllString(html, " ")
	html = reHeader.ReplaceAllString(html, " ")
	html = reTag.ReplaceAllString(html, " ")
	return strings.Join(strings.Fields(html), " ")
}
