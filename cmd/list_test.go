package cmd

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestListCatalogMatchesRegisteredTools는 cmd/list.go의 하드코딩된 카탈로그가
// 실제로 서버에 등록된 도구 이름과 정확히 일치하는지 검증한다.
// 과거 get_exchange_rate(실제: get_exchange_rates), get_real_estate_transactions
// (실제: get_real_estate_data)처럼 카탈로그가 실제 등록명과 드리프트된 적이 있어
// 이를 재발 방지하기 위한 테스트다.
func TestListCatalogMatchesRegisteredTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server := buildServer()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("서버 연결 실패: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "list-test-client", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("클라이언트 연결 실패: %v", err)
	}
	defer session.Close()

	var registered []string
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			t.Fatalf("도구 목록 조회 실패: %v", err)
		}
		registered = append(registered, tool.Name)
	}

	var cataloged []string
	for _, cat := range toolCategories {
		cataloged = append(cataloged, cat.tools...)
	}

	sort.Strings(registered)
	sort.Strings(cataloged)

	if len(registered) != len(cataloged) {
		t.Fatalf("등록된 도구 수(%d)와 카탈로그 수(%d)가 다름\n등록됨: %v\n카탈로그: %v",
			len(registered), len(cataloged), registered, cataloged)
	}
	for i := range registered {
		if registered[i] != cataloged[i] {
			t.Fatalf("카탈로그가 실제 등록명과 어긋남: 등록됨=%v, 카탈로그=%v", registered, cataloged)
		}
	}
}
