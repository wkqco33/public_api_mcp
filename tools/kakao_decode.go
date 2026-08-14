package tools

import "encoding/json"

// formatKakaoSearchResponse decodes each collection into its documented type.
// The collection switch is intentionally explicit: an unsupported collection
// cannot silently be accepted with a map[string]any fallback.
func formatKakaoSearchResponse(collection string, body []byte) (string, bool, error) {
	switch collection {
	case "web":
		var r kakaoResponse[kakaoWebDocument]
		if err := json.Unmarshal(body, &r); err != nil {
			return "", false, err
		}
		return formatKakaoWeb(r.Documents), len(r.Documents) == 0, nil
	case "blog":
		var r kakaoResponse[kakaoBlogDocument]
		if err := json.Unmarshal(body, &r); err != nil {
			return "", false, err
		}
		return formatKakaoBlog(r.Documents), len(r.Documents) == 0, nil
	case "cafe":
		var r kakaoResponse[kakaoCafeDocument]
		if err := json.Unmarshal(body, &r); err != nil {
			return "", false, err
		}
		return formatKakaoCafe(r.Documents), len(r.Documents) == 0, nil
	case "book":
		var r kakaoResponse[kakaoBookDocument]
		if err := json.Unmarshal(body, &r); err != nil {
			return "", false, err
		}
		return formatKakaoBook(r.Documents), len(r.Documents) == 0, nil
	case "image":
		var r kakaoResponse[kakaoImageDocument]
		if err := json.Unmarshal(body, &r); err != nil {
			return "", false, err
		}
		return formatKakaoImage(r.Documents), len(r.Documents) == 0, nil
	case "vclip":
		var r kakaoResponse[kakaoVideoDocument]
		if err := json.Unmarshal(body, &r); err != nil {
			return "", false, err
		}
		return formatKakaoVideo(r.Documents), len(r.Documents) == 0, nil
	default:
		return "", false, &unsupportedKakaoCollection{collection}
	}
}

type unsupportedKakaoCollection struct{ name string }

func (e *unsupportedKakaoCollection) Error() string {
	return "지원하지 않는 카카오 검색 컬렉션: " + e.name
}
