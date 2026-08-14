# AGENTS.md

이 프로젝트에 작업하는 모든 에이전트(인간·AI)를 위한 가이드입니다. 이 문서는 **TDD(Test-Driven Development)** 를 강제하기 위한 규칙과, 이 저장소의 아키텍처·관례를 설명합니다.

---

## 1. 프로젝트 요약

Go로 작성된 MCP(Model Context Protocol) 서버입니다. 한국 공공/민간 API를 MCP 도구로 노출합니다.

- **언어/런타임**: Go 1.26
- **도구 수**: 30개 (`public-api-mcp list`)
- **외부 API**: Kakao(Daum 검색/카카오맵), 환율, 유가, 주식, 부동산, 교통, 날씨, DART, 국립중앙도서관, Pixabay, Google News, 범용 HTTP

**이미 제거됨**: 네이버 OpenAPI, GitHub API. 다시 추가하지 마세요.

---

## 2. 가장 중요한 규칙: TDD

이 저장소는 **Red → Green → Refactor** 를 기본 작업 흐름으로 합니다.

모든 기능/버그 수정은 아래 순서를 따르세요.

1. **Red**: 먼저 실패하는 테스트를 작성한다.
2. **확인**: 그 테스트가 *올바른 이유로* 실패하는지 확인한다(실제 결함/미구현 때문에 실패하는지).
3. **Green**: 최소한의 구현으로 테스트를 통과시킨다.
4. **Refactor**: 중복 제거, 책임 분리, 가독성 개선.
5. **검증**: 전체 테스트 + vet + race 를 실행한다.
6. **커밋**: 작은 단위로 커밋한다.

> 코드를 먼저 짜고 나중에 테스트를 붙이는 것은 금지입니다.
> `go build`가 실패하더라도 **테스트 파일부터 작성**하고, 이후 구현으로 통과시킵니다.

### 커밋 단위

커밋은 기능·리팩터링·문서를 섞지 않고 하나의 의도만 담아 작게 쪼갭니다.

```text
test(kakao): add failing test for video page limit
feat(kakao): validate video page limit
refactor(kakao): extract search option normalization
docs: update README for new tool
```

---

## 3. 의존성 주입 원칙 (가장 중요한 아키텍처 규칙)

**외부 API 도구는 전역 상태(`config.X`)나 공유 HTTP transport에 직접 의존해서는 안 됩니다.**

각 도구는 생성자에서 의존성을 주입받습니다.

```go
// 카카오 예시
type KakaoClient struct {
    httpClient HTTPDoer
    baseURL    string
    apiKey     string
}

func NewKakaoClient(client HTTPDoer, baseURL, apiKey string) *KakaoClient
```

공통 규칙:

- `HTTPDoer` 인터페이스는 `Do(*http.Request) (*http.Response, error)` 하나의 메서드만 가집니다.
- `defaultXxxClient()` 는 프로덕션 기본값(실제 HTTP, 실제 base URL, `config.X` 키)을 조립합니다.
- **핸들러는 `defaultXxxClient()`를 만들지만**, 테스트는 `NewXxxClient(testServer.Client(), testServer.URL, "test-key")` 를 주입합니다.
- 테스트에서 `httpclient.Default.Transport`를 교체하거나 `config.X`를 직접 바꾸는 새 코드를 작성하지 마세요.
- 주입 가능한 항목: HTTP client, base URL, API 키, Clock(환율·날씨).

### HTTP 요청은 반드시 `httpclient.FetchBodyWithClient` 사용

```go
body, status, err := httpclient.FetchBodyWithClient(client.httpClient, req, maxAPIBodyBytes)
```

`httpclient.FetchBody(req, ...)`(전역 client 사용)는 새로 추가하지 마세요.

---

## 4. 테스트 구조

### 테스트 레벨 구분

| 레벨 | 대상 | 기법 |
|------|------|------|
| 순수 함수 | 검증, 포맷팅, 파싱, 시간 계산, 캐시 | 일반 단위 테스트 |
| HTTP adapter | API client | `httptest.Server` 주입 |
| MCP 핸들러 | `*CallToolResult` 변환 | fake client 주입 |
| 계약 | MCP 도구 등록 | MCP in-memory transport |

### 테스트 파일 관례

- 기능 파일과 같은 패키지(`package tools`)에 `*_test.go`.
- 순수 로직은 `xxx_test.go`, HTTP client는 `xxx_client_test.go` 처럼 역할별로 분리.
- 외부 API 응답은 **저장된 형태를 그대로** 흉내 낸 JSON/XML fixture 를 사용.
- 실시간 의존(`time.Now`)은 Clock을 주입하거나 경계 조건만 검증. `time.Sleep`을 테스트에 사용하지 마세요.
- 실제 네트워크/API 키를 쓰는 테스트는 기본 `go test`에 포함하지 마세요.

### 기존 스타일 참고

- `tools/kakao_test.go` — HTTP adapter + 검증
- `tools/kakao_validation_test.go`, `tools/kakao_formatter_test.go` — 순수 함수
- `tools/dart_corp_codes_test.go` — ZIP/XML 파싱
- `tools/weather_clock_test.go` — Clock 경계

---

## 5. 명령어

반드시 `task` 태스크를 사용합니다(수동으로 `go test` 만 실행하지 않음).

```bash
task check        # gofmt + go vet + race 테스트 (커밋 전 필수)
task test         # race detector 포함 전체 테스트
task test-cover   # 패키지별 커버리지 요약 + coverage.html
task build        # 바이너리 빌드
```

**커밋/PR 전에 반드시 `task check` 가 통과해야 합니다.**

커버리지는 최소한 **기존 수치 이하로 내리지 마세요.** 새로 추가/변경한 패키지나 파일은 80% 이상을 목표로 합니다.

---

## 6. 패키지·파일 배치

- 모든 외부 API 도구는 `tools/` 아래에 한 파일(+ 분리 파일)로 구성.
- 카카오처럼 책임이 크면 역할별로 분리: `kakao_model.go`, `kakao_validation.go`, `kakao_formatter.go`, `kakao_decode.go`.
- DART처럼 다단계 처리면 계층 분리: `dart_corp_codes.go`(파싱), `dart_corp_store.go`(파일 캐시), `dart_corp_download.go`(다운로드).
- 공통 HTTP 로직은 `httpclient/`, 공통 MCP 헬퍼는 `tools/util.go`(`toolText`, `toolError`, `withRecover`).

새 파일은 반드시 대응하는 테스트 파일과 함께 추가하세요.

---

## 7. 금지/주의 사항

### 금지

- `config.X`를 핸들러 안에서 직접 읽어 API 호출하기
- `httpclient.Default`를 핸들러에서 직접 `Do`/`FetchBody` 하기
- 테스트에서 전역 transport를 교체하는 새 코드 추가
- `map[string]any` 타입 단언으로 응답을 처리하는 새 코드 (명시적 struct 사용)
- `time.Sleep` 으로 시간 경과를 테스트
- 네이버/깃허브 도구 재추가
- 로그나 디버그 출력에 API 키·토큰 노출

### 주의

- 외부 API 응답은 실패(4xx/5xx), malformed JSON/XML, 빈 결과를 모두 테스트.
- 인증 실패(키 미설정), 429(쿼터), 403(권한)을 각각 명확한 오류로 처리.
- API 응답은 명시적 struct로 정의해 필드 오타를 컴파일/테스트 단계에서 잡는다.
- `withRecover`가 패닉을 복구해 주지만, 타입 단언보다는 타입 안전한 코드를 우선한다.

---

## 8. 새 API 도구를 추가할 때 체크리스트

```text
[ ] 입력 구조(struct) 정의
[ ] 입력 검증 및 기본값 (순수 함수 + 테스트)
[ ] HTTP client 구조체 + NewXxxClient + defaultXxxClient
[ ] 응답 모델(struct) 정의
[ ] HTTP 요청 → FetchBodyWithClient
[ ] 응답 파싱 + 오류 정규화
[ ] 출력 포맷터 (순수 함수 + 테스트)
[ ] MCP 핸들러 (+ WithClient 변형)
[ ] Register 함수에 도구 등록
[ ] cmd/list.go 카탈로그에 추가 (cmd/list_test.go가 일치 검증)
[ ] 성공/빈 결과/4xx/429/5xx/malformed 테스트
[ ] task check 통과 확인
[ ] README 도구 목록 갱신
```

---

## 9. 작업 시작 전 확인

- `git status` 로 작업 트리가 깨끗한지 확인.
- 최신 `main` 기준으로 브랜치 생성.
- 작업 범위를 README의 **테스트 및 TDD 개발**·**프로젝트 구조**와 대조해 현재 아키텍처 관례를 따르는지 확인.
