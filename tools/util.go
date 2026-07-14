package tools

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxAPIBodyBytes는 외부 공공 API 응답 바디를 읽을 때 적용하는 공통 상한이다.
const maxAPIBodyBytes = 5 * 1024 * 1024

func toolText(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func toolError(format string, args ...any) *mcp.CallToolResult {
	msg := fmt.Sprintf(format, args...)
	slog.Warn(msg)
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

// withRecover는 툴 핸들러를 감싸 패닉을 복구하고, 호출마다 이름·소요시간·성공여부를
// 표준화된 형태로 로깅한다. 핸들러 내부의 예기치 못한 패닉(예: 잘못된 타입 단언)이
// 서버 프로세스 전체를 죽이지 않고 사용자에게 에러 결과로 반환되도록 하는 방어선이다.
func withRecover[In any](name string, h func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, any, error)) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (result *mcp.CallToolResult, output any, err error) {
		start := time.Now()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("tool panic", "tool", name, "panic", r, "stack", string(debug.Stack()))
				result, output, err = toolError("내부 오류가 발생했습니다: %v", r), nil, nil
			}
		}()

		result, output, err = h(ctx, req, in)

		dur := time.Since(start)
		failed := err != nil || (result != nil && result.IsError)
		if failed {
			slog.Warn("tool 호출 실패", "tool", name, "ms", dur.Milliseconds())
		} else {
			slog.Info("tool 호출", "tool", name, "ms", dur.Milliseconds())
		}
		return
	}
}
