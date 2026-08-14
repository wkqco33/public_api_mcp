package tools

import (
	"archive/zip"
	"bytes"
	"testing"
)

func makeCorpCodeZip(t *testing.T, filename, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestParseCorpCodeZip(t *testing.T) {
	data := makeCorpCodeZip(t, "CORPCODE.xml", `<result><list><corp_name> 삼성전자 </corp_name><corp_code>00126380</corp_code></list><list><corp_name>빈코드</corp_name><corp_code></corp_code></list></result>`)
	codes, err := parseCorpCodeZip(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if codes["삼성전자"] != "00126380" || len(codes) != 1 {
		t.Fatalf("unexpected codes: %#v", codes)
	}
}

func TestParseCorpCodeZipErrors(t *testing.T) {
	if _, err := parseCorpCodeZip([]byte("not a zip")); err == nil {
		t.Fatal("invalid zip should fail")
	}
	data := makeCorpCodeZip(t, "README.txt", "no xml")
	if _, err := parseCorpCodeZip(data); err == nil {
		t.Fatal("zip without XML should fail")
	}
	data = makeCorpCodeZip(t, "CORPCODE.xml", "<broken")
	if _, err := parseCorpCodeZip(data); err == nil {
		t.Fatal("invalid XML should fail")
	}
}
