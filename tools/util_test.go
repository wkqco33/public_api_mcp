package tools

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToolText(t *testing.T) {
	result := toolText("hello")
	if result.IsError {
		t.Fatal("IsError가 true — 기대: false")
	}
	if len(result.Content) == 0 {
		t.Fatal("Content가 비어 있음")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Content[0]가 TextContent가 아님")
	}
	if tc.Text != "hello" {
		t.Fatalf("Text 불일치: got=%q, want=%q", tc.Text, "hello")
	}
}

func TestToolTextEmpty(t *testing.T) {
	result := toolText("")
	if result.IsError {
		t.Fatal("IsError가 true — 기대: false")
	}
	tc := result.Content[0].(*mcp.TextContent)
	if tc.Text != "" {
		t.Fatalf("빈 문자열 기대: got=%q", tc.Text)
	}
}

func TestToolError(t *testing.T) {
	result := toolError("오류: %s (%d)", "테스트", 42)
	if !result.IsError {
		t.Fatal("IsError가 false — 기대: true")
	}
	if len(result.Content) == 0 {
		t.Fatal("Content가 비어 있음")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Content[0]가 TextContent가 아님")
	}
	want := "오류: 테스트 (42)"
	if tc.Text != want {
		t.Fatalf("Text 불일치: got=%q, want=%q", tc.Text, want)
	}
}

func TestToolErrorIsError(t *testing.T) {
	result := toolError("단순 오류")
	if !result.IsError {
		t.Fatal("IsError가 false — 기대: true")
	}
	tc := result.Content[0].(*mcp.TextContent)
	if tc.Text != "단순 오류" {
		t.Fatalf("Text 불일치: got=%q", tc.Text)
	}
}
