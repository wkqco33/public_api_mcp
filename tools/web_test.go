package tools

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestExtractText(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{"일반 p 태그", "<p>Hello</p>", "Hello"},
		{"script 제거", "<script>alert(1)</script><p>본문</p>", "본문"},
		{"style 제거", "<style>.cls{color:red}</style><p>본문</p>", "본문"},
		{"nav 제거", "<nav>메뉴</nav><p>본문</p>", "본문"},
		{"footer 제거", "<p>본문</p><footer>푸터</footer>", "본문"},
		{"header 제거", "<header>헤더</header><p>본문</p>", "본문"},
		{"연속 공백 정리", "<p>Hello   World</p>", "Hello World"},
		{"빈 입력", "", ""},
		{"태그 없는 텍스트", "plain text", "plain text"},
		{"중첩 태그", "<div><p><span>내용</span></p></div>", "내용"},
		{
			"script와 본문 혼재",
			"<p>앞</p><script>var x=1;</script><p>뒤</p>",
			"앞 뒤",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractText(tt.html)
			if got != tt.want {
				t.Errorf("got=%q, want=%q", got, tt.want)
			}
		})
	}
}

func TestGetNewsCacheHit(t *testing.T) {
	// newsCache에 직접 저장 → getNews가 HTTP 요청 없이 캐시에서 반환하는지 확인
	topic := fmt.Sprintf("캐시테스트_%d", time.Now().UnixNano())
	cachedText := "- 캐시 뉴스 제목\n  URL: http://example.com/news\n"

	newsCache.Set(topic, cachedText, time.Minute)

	result, _, err := getNews(context.Background(), nil, getNewsInput{Topic: topic})
	if err != nil {
		t.Fatalf("getNews 오류: %v", err)
	}
	if result.IsError {
		t.Fatalf("에러 결과: %+v", result)
	}
	if len(result.Content) == 0 {
		t.Fatal("Content가 비어 있음")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Content[0]가 TextContent가 아님")
	}
	if tc.Text != cachedText {
		t.Errorf("캐시 값 불일치:\ngot=%q\nwant=%q", tc.Text, cachedText)
	}
}
