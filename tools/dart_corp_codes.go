package tools

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type dartCorpCodeList struct {
	Items []struct {
		Name string `xml:"corp_name"`
		Code string `xml:"corp_code"`
	} `xml:"list"`
}

// parseCorpCodeZip extracts and parses the XML member returned by DART.
// It is deliberately isolated from filesystem and HTTP concerns so malformed
// archives and filtering rules can be tested without environment setup.
func parseCorpCodeZip(data []byte) (map[string]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("ZIP 파싱 실패: %v", err)
	}
	var xmlBytes []byte
	for _, f := range zr.File {
		if strings.HasSuffix(strings.ToUpper(f.Name), ".XML") {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("XML 파일 열기 실패: %v", err)
			}
			xmlBytes, err = io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return nil, fmt.Errorf("XML 읽기 실패: %v", err)
			}
			break
		}
	}
	if len(xmlBytes) == 0 {
		return nil, fmt.Errorf("ZIP 안에 XML 파일이 없습니다")
	}
	var list dartCorpCodeList
	if err := xml.Unmarshal(xmlBytes, &list); err != nil {
		return nil, fmt.Errorf("XML 파싱 실패: %v", err)
	}
	codes := make(map[string]string, len(list.Items))
	for _, item := range list.Items {
		name, code := strings.TrimSpace(item.Name), strings.TrimSpace(item.Code)
		if name != "" && code != "" {
			codes[name] = code
		}
	}
	return codes, nil
}
