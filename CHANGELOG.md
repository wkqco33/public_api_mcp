# Changelog

이 프로젝트의 변경 사항을 기록합니다. 형식은 [Keep a Changelog](https://keepachangelog.com/ko/1.1.0/)를 따릅니다.

## [Unreleased]

### 공개 준비
- 저장소 공개용 준비: MIT 라이선스, CI, 기여 문서 추가
- 모듈 경로를 `github.com/wkqco33/public_api_mcp`로 변경
- Git 히스토리에서 GitHub 토큰 잔재 제거

### 리팩터링
- 전역 `config.X`/공유 HTTP transport 의존 제거, 외부 API 도구에 의존성 주입 적용
  - Kakao, 환율, 날씨, 주식, 교통, 유가, 부동산, 도서, Pixabay, DART
- 카카오 응답을 명시적 struct로 타입화하고 검증·포맷팅·디코딩 분리
- DART 기업코드 처리 계층 분리 (파싱 / 파일 캐시 / 다운로드 / 흐름)
- 날씨·환율에 Clock 주입
- HTTP 디버그 로그의 API 키 마스킹
- 명시한 `.env` 파일 오류를 서버 시작 시 반환

### 변경
- 네이버 OpenAPI 도구 제거 (서비스 종료)
- GitHub 도구 제거
- Kakao Daum 검색/카카오맵 Local 도구 7개 추가 (`kakao_search_*`, `kakao_search_places`)
- 총 도구 42개 → 30개

---

## 이전 버전 기록

`master` 히스토리를 참고하세요. 초기 스캐폴딩 이후 아래 기능이 추가되었습니다.

- 공공 API 도구 추가 (DART, 유가, 주식)
- HTTP/SSE 인증 (`MCP_AUTH_TOKEN`)
- 캐시 및 공유 HTTP 클라이언트
- OS 서비스 배포 (systemd / launchd)
