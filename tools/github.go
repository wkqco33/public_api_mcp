package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"wkqcosoft.com/public-api-mcp/config"
	"wkqcosoft.com/public-api-mcp/httpclient"
)

type githubListReposInput struct {
	Username string `json:"username,omitempty" jsonschema:"GitHub 사용자명. 비우면 내 저장소(토큰 필요), 지정하면 해당 유저의 public 저장소"`
}

type githubRepoStateInput struct {
	Repo  string `json:"repo" jsonschema:"저장소 이름 (owner/repo 형식)"`
	State string `json:"state,omitempty" jsonschema:"상태 필터: open, closed, all (기본: open)"`
}

type githubGetPRInput struct {
	Repo     string `json:"repo" jsonschema:"저장소 이름 (owner/repo 형식)"`
	PRNumber int    `json:"pr_number" jsonschema:"PR 번호"`
}

type githubCreateIssueInput struct {
	Repo  string `json:"repo" jsonschema:"저장소 이름 (owner/repo 형식)"`
	Title string `json:"title" jsonschema:"이슈 제목"`
	Body  string `json:"body,omitempty" jsonschema:"이슈 본문"`
}

type githubRepoOnlyInput struct {
	Repo string `json:"repo" jsonschema:"저장소 이름 (owner/repo 형식)"`
}

func RegisterGitHub(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{Name: "github_list_repos", Description: "GitHub 저장소 목록 조회. username 미지정 시 내 저장소(토큰 필요)."}, githubListRepos)
	mcp.AddTool(s, &mcp.Tool{Name: "github_list_prs", Description: "GitHub Pull Request 목록 조회."}, githubListPRs)
	mcp.AddTool(s, &mcp.Tool{Name: "github_get_pr", Description: "GitHub PR 상세 정보와 변경 파일 목록 조회."}, githubGetPR)
	mcp.AddTool(s, &mcp.Tool{Name: "github_list_issues", Description: "GitHub 이슈 목록 조회 (PR 제외)."}, githubListIssues)
	mcp.AddTool(s, &mcp.Tool{Name: "github_create_issue", Description: "GitHub 이슈 생성. GITHUB_TOKEN이 필요합니다."}, githubCreateIssue)
	mcp.AddTool(s, &mcp.Tool{Name: "github_workflow_runs", Description: "GitHub Actions 워크플로우 실행 내역 조회."}, githubWorkflowRuns)
}

func ghHeaders() map[string]string {
	h := map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
	}
	if config.GitHubToken != "" {
		h["Authorization"] = "Bearer " + config.GitHubToken
	}
	return h
}

func ghGet(apiPath string) ([]byte, int, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com"+apiPath, nil)
	for k, v := range ghHeaders() {
		req.Header.Set(k, v)
	}
	res, err := httpclient.Default.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 5*1024*1024))
	if err != nil {
		return nil, 0, err
	}
	return body, res.StatusCode, nil
}

func githubListRepos(_ context.Context, _ *mcp.CallToolRequest, in githubListReposInput) (*mcp.CallToolResult, any, error) {
	if in.Username == "" && config.GitHubToken == "" {
		return toolError("내 저장소 조회는 GITHUB_TOKEN이 필요합니다"), nil, nil
	}
	path := "/user/repos?per_page=30&sort=updated"
	if in.Username != "" {
		path = fmt.Sprintf("/users/%s/repos?per_page=30&sort=updated", in.Username)
	}

	body, code, err := ghGet(path)
	if err != nil {
		return toolError("GitHub API 요청 실패: %v", err), nil, nil
	}
	if code != http.StatusOK {
		return toolError("GitHub API 오류 [%d]: %s", code, string(body)), nil, nil
	}

	var repos []map[string]any
	if err := json.Unmarshal(body, &repos); err != nil {
		return toolError("응답 파싱 실패: %v", err), nil, nil
	}
	if len(repos) == 0 {
		return toolText("저장소 없음"), nil, nil
	}

	var sb strings.Builder
	for _, r := range repos {
		lang, _ := r["language"].(string)
		if lang == "" {
			lang = "-"
		}
		desc, _ := r["description"].(string)
		stars, _ := r["stargazers_count"].(float64)
		fmt.Fprintf(&sb, "%s ⭐%.0f [%s] %s\n", r["full_name"], stars, lang, desc)
	}
	return toolText(sb.String()), nil, nil
}

func githubListPRs(_ context.Context, _ *mcp.CallToolRequest, in githubRepoStateInput) (*mcp.CallToolResult, any, error) {
	state := in.State
	if state == "" {
		state = "open"
	}
	body, code, err := ghGet(fmt.Sprintf("/repos/%s/pulls?state=%s&per_page=20", in.Repo, state))
	if err != nil {
		return toolError("GitHub API 요청 실패: %v", err), nil, nil
	}
	if code != http.StatusOK {
		return toolError("GitHub API 오류 [%d]: %s", code, string(body)), nil, nil
	}

	var prs []map[string]any
	if err := json.Unmarshal(body, &prs); err != nil {
		return toolError("응답 파싱 실패: %v", err), nil, nil
	}
	if len(prs) == 0 {
		return toolText(fmt.Sprintf("PR 없음 (state=%s)", state)), nil, nil
	}

	var sb strings.Builder
	for _, p := range prs {
		user, _ := p["user"].(map[string]any)
		login, _ := user["login"].(string)
		num, _ := p["number"].(float64)
		fmt.Fprintf(&sb, "#%.0f [%s] %s — @%s\n", num, p["state"], p["title"], login)
	}
	return toolText(sb.String()), nil, nil
}

func githubGetPR(_ context.Context, _ *mcp.CallToolRequest, in githubGetPRInput) (*mcp.CallToolResult, any, error) {
	prBody, code, err := ghGet(fmt.Sprintf("/repos/%s/pulls/%d", in.Repo, in.PRNumber))
	if err != nil {
		return toolError("GitHub API 요청 실패: %v", err), nil, nil
	}
	if code != http.StatusOK {
		return toolError("GitHub API 오류 [%d]: %s", code, string(prBody)), nil, nil
	}

	filesBody, code2, err := ghGet(fmt.Sprintf("/repos/%s/pulls/%d/files?per_page=30", in.Repo, in.PRNumber))
	if err != nil || code2 != http.StatusOK {
		return toolError("파일 목록 조회 실패"), nil, nil
	}

	var pr map[string]any
	if err := json.Unmarshal(prBody, &pr); err != nil {
		return toolError("PR 파싱 실패: %v", err), nil, nil
	}
	var files []map[string]any
	if err := json.Unmarshal(filesBody, &files); err != nil {
		return toolError("파일 목록 파싱 실패: %v", err), nil, nil
	}

	user, _ := pr["user"].(map[string]any)
	login, _ := user["login"].(string)
	head, _ := pr["head"].(map[string]any)
	base, _ := pr["base"].(map[string]any)
	num, _ := pr["number"].(float64)
	commits, _ := pr["commits"].(float64)
	changed, _ := pr["changed_files"].(float64)
	additions, _ := pr["additions"].(float64)
	deletions, _ := pr["deletions"].(float64)
	body, _ := pr["body"].(string)
	if body == "" {
		body = "(없음)"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "PR #%.0f: %s\n상태: %s | 작성자: @%s\n브랜치: %s → %s\n커밋: %.0f | 변경파일: %.0f | +%.0f/-%.0f\n\n변경 파일:\n",
		num, pr["title"], pr["state"], login, head["ref"], base["ref"], commits, changed, additions, deletions)
	for _, f := range files {
		add, _ := f["additions"].(float64)
		del, _ := f["deletions"].(float64)
		status, _ := f["status"].(string)
		filename, _ := f["filename"].(string)
		fmt.Fprintf(&sb, "  %-8s +%.0f/-%.0f %s\n", status, add, del, filename)
	}
	fmt.Fprintf(&sb, "\n설명:\n%s", body)

	return toolText(sb.String()), nil, nil
}

func githubListIssues(_ context.Context, _ *mcp.CallToolRequest, in githubRepoStateInput) (*mcp.CallToolResult, any, error) {
	state := in.State
	if state == "" {
		state = "open"
	}
	body, code, err := ghGet(fmt.Sprintf("/repos/%s/issues?state=%s&per_page=20", in.Repo, state))
	if err != nil {
		return toolError("GitHub API 요청 실패: %v", err), nil, nil
	}
	if code != http.StatusOK {
		return toolError("GitHub API 오류 [%d]: %s", code, string(body)), nil, nil
	}

	var issues []map[string]any
	if err := json.Unmarshal(body, &issues); err != nil {
		return toolError("응답 파싱 실패: %v", err), nil, nil
	}

	var sb strings.Builder
	count := 0
	for _, issue := range issues {
		// PR은 제외
		if _, isPR := issue["pull_request"]; isPR {
			continue
		}
		user, _ := issue["user"].(map[string]any)
		login, _ := user["login"].(string)
		num, _ := issue["number"].(float64)
		fmt.Fprintf(&sb, "#%.0f [%s] %s — @%s\n", num, issue["state"], issue["title"], login)
		count++
	}
	if count == 0 {
		return toolText(fmt.Sprintf("이슈 없음 (state=%s)", state)), nil, nil
	}
	return toolText(sb.String()), nil, nil
}

func githubCreateIssue(_ context.Context, _ *mcp.CallToolRequest, in githubCreateIssueInput) (*mcp.CallToolResult, any, error) {
	if config.GitHubToken == "" {
		return toolError("GITHUB_TOKEN이 설정되지 않았습니다"), nil, nil
	}

	payload, _ := json.Marshal(map[string]string{"title": in.Title, "body": in.Body})
	req, _ := http.NewRequest("POST", fmt.Sprintf("https://api.github.com/repos/%s/issues", in.Repo), strings.NewReader(string(payload)))
	for k, v := range ghHeaders() {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := httpclient.Default.Do(req)
	if err != nil {
		return toolError("GitHub API 요청 실패: %v", err), nil, nil
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1*1024*1024))
	if err != nil {
		return toolError("응답 읽기 실패: %v", err), nil, nil
	}
	if res.StatusCode != http.StatusCreated {
		return toolError("이슈 생성 실패 [%d]: %s", res.StatusCode, string(body)), nil, nil
	}

	var issue map[string]any
	if err := json.Unmarshal(body, &issue); err != nil {
		return toolError("이슈 응답 파싱 실패: %v", err), nil, nil
	}
	num, _ := issue["number"].(float64)
	htmlURL, _ := issue["html_url"].(string)
	return toolText(fmt.Sprintf("이슈 생성 완료: #%.0f\nURL: %s", num, htmlURL)), nil, nil
}

func githubWorkflowRuns(_ context.Context, _ *mcp.CallToolRequest, in githubRepoOnlyInput) (*mcp.CallToolResult, any, error) {
	body, code, err := ghGet(fmt.Sprintf("/repos/%s/actions/runs?per_page=10", in.Repo))
	if err != nil {
		return toolError("GitHub API 요청 실패: %v", err), nil, nil
	}
	if code != http.StatusOK {
		return toolError("GitHub API 오류 [%d]: %s", code, string(body)), nil, nil
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return toolError("응답 파싱 실패: %v", err), nil, nil
	}

	runs, _ := data["workflow_runs"].([]any)
	if len(runs) == 0 {
		return toolText("워크플로우 실행 내역 없음"), nil, nil
	}

	var sb strings.Builder
	for _, raw := range runs {
		run, _ := raw.(map[string]any)
		conclusion, _ := run["conclusion"].(string)
		if conclusion == "" {
			conclusion = "진행중"
		}
		createdAt, _ := run["created_at"].(string)
		fmt.Fprintf(&sb, "[%s] %s — %s (%s)\n", run["status"], run["name"], conclusion, createdAt)
	}
	return toolText(sb.String()), nil, nil
}
