package tools

import (
	"path/filepath"
	"testing"
	"time"
)

func TestFileDARTCorpCodeStoreSaveLoad(t *testing.T) {
	store := newFileDARTCorpCodeStore(filepath.Join(t.TempDir(), "nested", "codes.json"), time.Hour)
	want := map[string]string{"삼성전자": "00126380"}
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	got, found, err := store.Load()
	if err != nil || !found || got["삼성전자"] != want["삼성전자"] {
		t.Fatalf("got=%#v found=%v err=%v", got, found, err)
	}
}

func TestFileDARTCorpCodeStoreExpiration(t *testing.T) {
	store := newFileDARTCorpCodeStore(filepath.Join(t.TempDir(), "codes.json"), time.Hour)
	if err := store.Save(map[string]string{"기업": "123"}); err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	if _, found, err := store.Load(); err != nil || found {
		t.Fatalf("expired cache: found=%v err=%v", found, err)
	}
}

func TestFileDARTCorpCodeStoreMissing(t *testing.T) {
	store := newFileDARTCorpCodeStore(filepath.Join(t.TempDir(), "missing.json"), time.Hour)
	if _, found, err := store.Load(); err != nil || found {
		t.Fatalf("missing cache: found=%v err=%v", found, err)
	}
}
