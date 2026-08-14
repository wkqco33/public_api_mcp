package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExplicitEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.env")
	if err := os.WriteFile(path, []byte("KAKAO_REST_API_KEY=from-file\nMCP_AUTH_TOKEN=auth\n"), 0600); err != nil {
		t.Fatal(err)
	}
	oldKakao, oldAuth := os.Getenv("KAKAO_REST_API_KEY"), os.Getenv("MCP_AUTH_TOKEN")
	t.Cleanup(func() { _ = os.Setenv("KAKAO_REST_API_KEY", oldKakao); _ = os.Setenv("MCP_AUTH_TOKEN", oldAuth) })
	_ = os.Unsetenv("KAKAO_REST_API_KEY")
	_ = os.Unsetenv("MCP_AUTH_TOKEN")
	if err := Load(path); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if KakaoRESTAPIKey != "from-file" || MCPAuthToken != "auth" {
		t.Fatalf("loaded values = %q, %q", KakaoRESTAPIKey, MCPAuthToken)
	}
}

func TestLoadExplicitMissingFileReturnsError(t *testing.T) {
	if err := Load(filepath.Join(t.TempDir(), "missing.env")); err == nil {
		t.Fatal("missing explicit env file should return an error")
	}
}
