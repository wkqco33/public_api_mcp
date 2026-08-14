package tools

import (
	"context"
	"fmt"
	"net/http"

	"wkqcosoft.com/public-api-mcp/httpclient"
)

type dartCorpCodeDownloader interface {
	Download(ctx context.Context) ([]byte, error)
}

type httpDARTCorpCodeDownloader struct {
	client *DARTClient
}

// Download fetches the ZIP archive of corporate codes from DART. It keeps the
// API-key and base URL on the injected client so the downloader can be tested
// against a local HTTP server without real credentials or network access.
func (d *httpDARTCorpCodeDownloader) Download(ctx context.Context) ([]byte, error) {
	if d.client.apiKey == "" {
		return nil, fmt.Errorf("OPENDART_API_KEY가 설정되지 않았습니다")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", d.client.baseURL+"/api/corpCode.xml", nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}
	q := req.URL.Query()
	q.Set("crtfc_key", d.client.apiKey)
	req.URL.RawQuery = q.Encode()

	body, _, err := httpclient.FetchBodyWithClient(d.client.httpClient, req, 50*1024*1024) // 50MB 제한
	if err != nil {
		return nil, fmt.Errorf("기업코드 다운로드 실패: %w", err)
	}
	return body, nil
}
