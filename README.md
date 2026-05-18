# public-api-mcp

한국 공공 API 37개를 MCP(Model Context Protocol) 서버로 노출합니다. Claude Desktop, Cline 등 MCP 클라이언트에서 바로 사용할 수 있습니다.

---

## 도구 목록 (37개)

| 카테고리 | 도구 | 필요 API 키 |
|----------|------|-------------|
| 네이버 검색 | `naver_search_blog`, `naver_search_news`, `naver_search_cafe`, `naver_search_book`, `naver_search_encyc`, `naver_search_kin`, `naver_search_local`, `naver_search_webkr`, `naver_search_image`, `naver_search_shop`, `naver_search_doc`, `naver_check_adult`, `naver_convert_errata` | `NAVER_CLIENT_ID`, `NAVER_CLIENT_SECRET` |
| GitHub | `github_list_repos`, `github_list_prs`, `github_get_pr`, `github_list_issues`, `github_create_issue`, `github_workflow_runs` | `GITHUB_TOKEN` |
| 환율 | `get_exchange_rates` | `EXCHANGE_API_KEY` |
| 유가 | `get_fuel_avg_price`, `get_fuel_sido_price` | `OPINET_API_KEY` |
| 주식 | `get_stock_price`, `get_stock_analysis_data` | `PUBLIC_DATA_PORTAL_API_KEY` |
| 부동산 | `get_real_estate_transactions` | `PUBLIC_DATA_PORTAL_API_KEY` |
| 교통 | `search_bus_stop`, `get_bus_arrival`, `get_subway_arrival` | `PUBLIC_DATA_PORTAL_API_KEY`, `SEOUL_OPEN_API_KEY` |
| 날씨 | `get_current_weather`, `get_weather_forecast` | `KMA_WEATHER_API_KEY` |
| DART 공시 | `search_dart_company`, `get_dart_disclosures` | `OPENDART_API_KEY` |
| 웹 | `get_news`, `fetch_url` | 없음 |
| 도서 | `get_recommended_books` | `NATIONAL_LIBRARY_API_KEY` |
| 이미지 | `search_pixabay_images` | `PIXABAY_API_KEY` |
| HTTP | `http_request` | 없음 |

---

## 시작하기

### 사전 요구사항

- Go 1.21 이상

### 빌드

```bash
git clone <repo-url>
cd public_api_mcp
make          # 빌드
make install  # /usr/local/bin에 설치 (sudo 필요할 수 있음)
```

사용자 디렉토리에 설치:

```bash
make install PREFIX=~/.local
```

직접 빌드:

```bash
go build -o public-api-mcp .
```

---

## API 키 설정

### .env 파일 생성

```bash
cp .env.example .env
```

`.env` 파일을 열고 사용할 API 키를 채워 넣습니다. 사용하지 않는 도구의 키는 비워 두어도 됩니다.

### API 키 발급 방법

| 환경변수 | 서비스 | 발급 URL |
|----------|--------|----------|
| `NAVER_CLIENT_ID`<br>`NAVER_CLIENT_SECRET` | 네이버 OpenAPI | https://developers.naver.com/apps/#/register |
| `EXCHANGE_API_KEY` | 한국수출입은행 환율 | https://www.koreaexim.go.kr/ir/HPHKIR020M01?apino=2 |
| `PUBLIC_DATA_PORTAL_API_KEY` | 공공데이터포털 (주식, 부동산, 버스) | https://www.data.go.kr — 회원가입 후 원하는 API 신청 |
| `SEOUL_OPEN_API_KEY` | 서울 열린데이터광장 (지하철) | https://data.seoul.go.kr |
| `KMA_WEATHER_API_KEY` | 기상청 단기예보 | https://www.data.go.kr — "기상청_단기예보" 검색 후 신청 |
| `OPENDART_API_KEY` | 금융감독원 DART | https://opendart.fss.or.kr |
| `OPINET_API_KEY` | 한국석유공사 오피넷 | https://www.opinet.co.kr/user/custapi/custApiInfo.do |
| `GITHUB_TOKEN` | GitHub Personal Access Token | https://github.com/settings/tokens — `repo` 스코프 필요 |
| `NATIONAL_LIBRARY_API_KEY` | 국립중앙도서관 | https://www.nl.go.kr/NL/contents/N31101030000.do |
| `PIXABAY_API_KEY` | Pixabay 이미지 | https://pixabay.com/api/docs/ |

### 설정 확인

```bash
# API 키 현황 출력 (값은 마스킹)
./public-api-mcp config show

# 미설정 키 목록 경고
./public-api-mcp config validate
```

---

## 실행 방법

### stdio (Claude Desktop / Cline 등 MCP 클라이언트)

```bash
./public-api-mcp serve stdio
```

### HTTP/SSE

```bash
./public-api-mcp serve http --addr :8080
```

### 커스텀 .env 파일 경로 지정

```bash
./public-api-mcp --env /etc/mcp/.env serve stdio
```

### 도구 목록 확인

```bash
./public-api-mcp list
```

---

## 클라이언트별 설정

`config-examples/` 디렉토리에 플랫폼별 설정 예시 파일이 있습니다. `/절대경로/public-api-mcp` 부분을 실제 바이너리 경로로 교체해서 사용하세요.

### Claude Desktop

설정 파일 위치:
- Linux: `~/.config/claude/claude_desktop_config.json`
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "public-api-mcp": {
      "command": "/절대경로/public-api-mcp",
      "args": ["serve", "stdio"]
    }
  }
}
```

예시 파일: `config-examples/claude-desktop.json`

### VS Code (GitHub Copilot)

VS Code 1.99 이상. 워크스페이스 루트에 `.vscode/mcp.json` 파일을 생성합니다.

```json
{
  "servers": {
    "public-api-mcp": {
      "type": "stdio",
      "command": "/절대경로/public-api-mcp",
      "args": ["serve", "stdio"]
    }
  }
}
```

예시 파일: `config-examples/vscode.json`

### Cursor

설정 파일 위치: `~/.cursor/mcp.json` (전역) 또는 `.cursor/mcp.json` (프로젝트별)

```json
{
  "mcpServers": {
    "public-api-mcp": {
      "command": "/절대경로/public-api-mcp",
      "args": ["serve", "stdio"]
    }
  }
}
```

예시 파일: `config-examples/cursor.json`

### Gemini CLI

설정 파일 위치: `~/.gemini/settings.json`

```json
{
  "mcpServers": {
    "public-api-mcp": {
      "command": "/절대경로/public-api-mcp",
      "args": ["serve", "stdio"]
    }
  }
}
```

예시 파일: `config-examples/gemini-cli.json`

### Cline (VS Code 확장)

VS Code `settings.json`에 추가합니다 (`Ctrl+Shift+P` → "Open User Settings JSON").

```json
{
  "cline.mcpServers": {
    "public-api-mcp": {
      "command": "/절대경로/public-api-mcp",
      "args": ["serve", "stdio"],
      "disabled": false,
      "alwaysAllow": []
    }
  }
}
```

예시 파일: `config-examples/cline.json`

### env 파일 경로를 별도로 지정하는 경우

`.env` 파일이 바이너리와 다른 위치에 있을 때 `--env` 플래그로 경로를 지정합니다. 위 모든 플랫폼에서 동일하게 적용됩니다.

```json
"args": ["--env", "/절대경로/.env", "serve", "stdio"]
```

---

## 디버그 모드

`--debug` 플래그를 추가하면 모든 HTTP 요청의 URL, 상태코드, 소요시간과 캐시 hit/miss 정보를 stderr에 출력합니다.

```bash
./public-api-mcp --debug serve http --addr :8080
```

출력 예시:

```
time=2026-05-18T10:00:00+09:00 level=INFO msg="HTTP MCP 서버 시작" addr=:8080
time=2026-05-18T10:00:01+09:00 level=DEBUG msg=HTTP method=GET url="https://openapi.naver.com/..." status=200 ms=143
time=2026-05-18T10:00:02+09:00 level=DEBUG msg="환율 캐시 miss" date=20260518
time=2026-05-18T10:00:03+09:00 level=WARN msg="네이버 API 오류 [429]: rate limit exceeded"
```

기본 모드(--debug 없음)에서는 WARN 이상만 출력되며, stdio 전송 시 stdout은 MCP JSON-RPC 전용으로 유지됩니다.

---

## 프로젝트 구조

```
public_api_mcp/
├── main.go              # 진입점 (cmd.Execute() 호출)
├── config-examples/     # 클라이언트별 MCP 설정 예시
│   ├── claude-desktop.json
│   ├── vscode.json
│   ├── cursor.json
│   ├── gemini-cli.json
│   └── cline.json
├── cmd/
│   ├── root.go          # RootCmd, 전역 플래그 (--env, --debug)
│   ├── serve.go         # serve stdio | serve http
│   ├── list.go          # list
│   └── config.go        # config show | config validate
├── config/
│   └── config.go        # 환경변수 로딩 (Load 함수)
├── cache/
│   └── cache.go         # 제네릭 TTL 캐시
├── httpclient/
│   └── client.go        # 공유 HTTP 클라이언트 (debug RoundTripper)
└── tools/
    ├── util.go           # 공유 헬퍼 (toolText, toolError)
    ├── naver.go          # 네이버 OpenAPI (13개)
    ├── github.go         # GitHub API (6개)
    ├── exchange.go       # 환율 (1개)
    ├── fuel.go           # 유가 (2개)
    ├── stock.go          # 주식 (2개)
    ├── real_estate.go    # 부동산 (1개)
    ├── transport.go      # 버스/지하철 (3개)
    ├── weather.go        # 날씨 (2개)
    ├── dart.go           # DART 공시 (2개)
    ├── web.go            # 뉴스/URL 가져오기 (2개)
    ├── book.go           # 도서 (1개)
    ├── pixabay.go        # 이미지 (1개)
    └── http_request.go   # 범용 HTTP (1개)
```
