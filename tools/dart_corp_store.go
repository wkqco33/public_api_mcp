package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type dartCorpCodeStore interface {
	Load() (codes map[string]string, found bool, err error)
	Save(map[string]string) error
}

type fileDARTCorpCodeStore struct {
	path string
	ttl  time.Duration
	now  func() time.Time
}

func newFileDARTCorpCodeStore(path string, ttl time.Duration) *fileDARTCorpCodeStore {
	return &fileDARTCorpCodeStore{path: path, ttl: ttl, now: time.Now}
}

func (s *fileDARTCorpCodeStore) Load() (map[string]string, bool, error) {
	info, err := os.Stat(s.path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("기업코드 캐시 확인 실패: %w", err)
	}
	if s.now().Sub(info.ModTime()) >= s.ttl {
		return nil, false, nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, false, fmt.Errorf("기업코드 캐시 읽기 실패: %w", err)
	}
	var codes map[string]string
	if err := json.Unmarshal(data, &codes); err != nil {
		return nil, false, fmt.Errorf("기업코드 캐시 파싱 실패: %w", err)
	}
	return codes, true, nil
}

func (s *fileDARTCorpCodeStore) Save(codes map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("기업코드 캐시 디렉터리 생성 실패: %w", err)
	}
	data, err := json.Marshal(codes)
	if err != nil {
		return fmt.Errorf("기업코드 캐시 직렬화 실패: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("기업코드 캐시 저장 실패: %w", err)
	}
	return nil
}
