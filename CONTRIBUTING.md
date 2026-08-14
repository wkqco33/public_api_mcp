# 기여 가이드

프로젝트에 기여해 주셔서 감사합니다. 이 저장소는 **TDD(Test-Driven Development)** 를 기본 작업 방식으로 합니다.

시작 전에 다음 문서를 먼저 읽어주세요.

- **`AGENTS.md`** — 에이전트(인간·AI)를 위한 TDD 규칙과 아키텍처 가이드
- **`README.md`** — 프로젝트 개요 및 사용법
- **`CODE_OF_CONDUCT.md`** — 참여 행동 강령

---

## 개발 환경

- Go 1.26 이상
- [Task](https://taskfile.dev/) (`brew install go-task` 또는 `go install github.com/go-task/task/v3/cmd/task@latest`)

## 빌드 및 테스트

```bash
task build        # 바이너리 빌드
task check        # gofmt + go vet + race 테스트 (커밋 전 필수)
task test         # race detector 포함 전체 테스트
task test-cover   # 커버리지 요약 + HTML 리포트
```

커밋/PR 전에 **반드시 `task check` 가 통과**해야 합니다.

---

## 작업 흐름

1. `main`을 최신으로 유지하고 기능 브랜치를 생성합니다.
2. 작업은 **Red → Green → Refactor** 순서로 진행합니다.
   - 먼저 실패하는 테스트를 작성한다.
   - 최소 구현으로 통과시킨다.
   - 중복과 결합을 정리한다.
3. `task check` 로 전체 검증합니다.
4. 작은 단위로 커밋합니다.

### 커밋 메시지 관례

```text
test(kakao): add failing test for video page limit
feat(kakao): validate video page limit
refactor(kakao): extract search option normalization
fix(weather): correct base-time boundary at midnight
docs: update README for new tool
```

하나의 커밋은 하나의 의도를 담아야 합니다. 기능·리팩터링·문서를 섞지 마세요.

---

## 코드 규칙

- 외부 API 도구는 `NewXxxClient(httpDoer, baseURL, apiKey)` 생성자로 의존성을 주입받습니다.
- 핸들러에서 `config.X`나 `httpclient.Default`를 직접 사용하지 마세요.
- HTTP 요청은 `httpclient.FetchBodyWithClient` 를 사용하세요.
- API 응답은 `map[string]any`가 아닌 명시적 struct 로 처리하세요.
- 테스트에서 실제 네트워크/API 키를 쓰지 마세요. `httptest.Server` 를 주입하세요.
- 새 기능 파일은 대응하는 테스트 파일과 함께 추가하세요.

자세한 내용은 **`AGENTS.md`** 를 참고하세요.

---

## 새 도구 추가 체크리스트

```text
[ ] 입력 구조 정의
[ ] 입력 검증 및 기본값 (순수 함수 + 테스트)
[ ] HTTP client + NewXxxClient + defaultXxxClient
[ ] 응답 모델(struct) 정의
[ ] HTTP 요청 → FetchBodyWithClient
[ ] 응답 파싱 + 오류 정규화
[ ] 출력 포맷터 (순수 함수 + 테스트)
[ ] MCP 핸들러 (+ WithClient 변형)
[ ] Register 함수에 도구 등록
[ ] cmd/list.go 카탈로그 추가
[ ] 성공/빈 결과/4xx/429/5xx/malformed 테스트
[ ] task check 통과
[ ] README 도구 목록 갱신
```

---

## 버그 리포트 / 기능 제안

- 이슈는 [GitHub Issues](../../issues) 에 등록합니다.
- 제안 시 해당 도구 이름과 요청/응답 예시를 함께 주시면 처리에 도움이 됩니다.
