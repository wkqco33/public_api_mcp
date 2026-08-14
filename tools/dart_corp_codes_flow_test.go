package tools

import (
	"context"
	"errors"
	"testing"
)

type fakeDARTStore struct {
	loadCodes map[string]string
	loadFound bool
	loadErr   error
	saved     map[string]string
}

func (f *fakeDARTStore) Load() (map[string]string, bool, error) {
	return f.loadCodes, f.loadFound, f.loadErr
}
func (f *fakeDARTStore) Save(codes map[string]string) error { f.saved = codes; return nil }

type fakeDARTDownloader struct {
	zipData []byte
	err     error
}

func (f *fakeDARTDownloader) Download(ctx context.Context) ([]byte, error) { return f.zipData, f.err }

func TestLoadCorpCodesUsesFileCache(t *testing.T) {
	setupTestCorpCodes(t, nil)
	store := &fakeDARTStore{loadCodes: map[string]string{"캐시기업": "11111111"}, loadFound: true}
	downloader := &fakeDARTDownloader{zipData: []byte("should-not-be-called")}
	codes, err := loadCorpCodesWithDependencies(store, downloader)
	if err != nil {
		t.Fatal(err)
	}
	if codes["캐시기업"] != "11111111" {
		t.Fatalf("cache not used: %#v", codes)
	}
}

func TestLoadCorpCodesDownloadsAndSaves(t *testing.T) {
	setupTestCorpCodes(t, nil)
	store := &fakeDARTStore{}
	zip := makeCorpCodeZip(t, "CORPCODE.xml", `<result><list><corp_name>다운로드기업</corp_name><corp_code>22222222</corp_code></list></result>`)
	downloader := &fakeDARTDownloader{zipData: zip}
	codes, err := loadCorpCodesWithDependencies(store, downloader)
	if err != nil {
		t.Fatal(err)
	}
	if codes["다운로드기업"] != "22222222" {
		t.Fatalf("download result missing: %#v", codes)
	}
	if store.saved["다운로드기업"] != "22222222" {
		t.Fatalf("codes not saved: %#v", store.saved)
	}
}

func TestLoadCorpCodesPropagatesDownloadError(t *testing.T) {
	setupTestCorpCodes(t, nil)
	downloader := &fakeDARTDownloader{err: errors.New("download failed")}
	if _, err := loadCorpCodesWithDependencies(&fakeDARTStore{}, downloader); err == nil {
		t.Fatal("download error should propagate")
	}
}
