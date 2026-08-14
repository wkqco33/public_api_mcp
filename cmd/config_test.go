package cmd

import "testing"

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "(미설정)"},
		{"abc", "****"},
		{"abcdefgh", "****"},                            // 8자
		{"abcdefghi", "abcd****fghi"},                   // 9자
		{"1234567890123456", "1234****3456"},            // 16자
		{"kakao-rest-key-example-1234", "kaka****1234"}, // Kakao REST key
	}

	for _, tt := range tests {
		got := maskKey(tt.input)
		if got != tt.want {
			t.Errorf("maskKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
