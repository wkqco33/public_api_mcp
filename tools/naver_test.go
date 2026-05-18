package tools

import "testing"

func TestStrVal(t *testing.T) {
	m := map[string]any{
		"name":  "삼성전자",
		"count": 42,
		"empty": "",
	}

	tests := []struct {
		key  string
		want string
	}{
		{"name", "삼성전자"},
		{"empty", ""},
		{"notexist", ""},
		{"count", ""},  // int → string 캐스팅 실패 → 빈 문자열
	}

	for _, tt := range tests {
		got := strVal(m, tt.key)
		if got != tt.want {
			t.Errorf("strVal(m, %q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<b>굵게</b>", "굵게"},
		{"&quot;quoted&quot;", `"quoted"`},
		{"A&amp;B", "A&B"},
		{"&lt;tag&gt;", "<tag>"},
		{"<b>A&amp;B</b>", "A&B"},
		{"plain text", "plain text"},
		{"", ""},
		{"<b></b>", ""},
		{"Hello &lt;world&gt; &amp; &quot;friends&quot;", `Hello <world> & "friends"`},
	}

	for _, tt := range tests {
		got := stripHTMLTags(tt.input)
		if got != tt.want {
			t.Errorf("stripHTMLTags(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
