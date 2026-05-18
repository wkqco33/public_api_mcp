package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var toolCategories = []struct {
	category string
	tools    []string
}{
	{"naver (네이버 OpenAPI)", []string{
		"naver_search_blog", "naver_search_news", "naver_search_cafe",
		"naver_search_book", "naver_search_encyc", "naver_search_kin",
		"naver_search_local", "naver_search_webkr", "naver_search_image",
		"naver_search_shop", "naver_search_doc", "naver_check_adult",
		"naver_convert_errata",
	}},
	{"github (GitHub API)", []string{
		"github_list_repos", "github_list_prs", "github_get_pr",
		"github_list_issues", "github_create_issue", "github_workflow_runs",
	}},
	{"exchange (한국수출입은행)", []string{
		"get_exchange_rate",
	}},
	{"fuel (오피넷)", []string{
		"get_fuel_avg_price", "get_fuel_sido_price",
	}},
	{"stock (공공데이터포털)", []string{
		"get_stock_price", "get_stock_analysis_data",
	}},
	{"real_estate (공공데이터포털)", []string{
		"get_real_estate_transactions",
	}},
	{"transport (공공데이터포털 + 서울시)", []string{
		"search_bus_stop", "get_bus_arrival", "get_subway_arrival",
	}},
	{"weather (기상청)", []string{
		"get_current_weather", "get_weather_forecast",
	}},
	{"dart (금융감독원 DART)", []string{
		"search_dart_company", "get_dart_disclosures",
	}},
	{"web", []string{
		"get_news", "fetch_url",
	}},
	{"book (국립중앙도서관)", []string{
		"get_recommended_books",
	}},
	{"pixabay", []string{
		"search_pixabay_images",
	}},
	{"http_request (범용 HTTP)", []string{
		"http_request",
	}},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "등록된 MCP 도구 목록 출력",
	Run: func(cmd *cobra.Command, args []string) {
		total := 0
		for _, cat := range toolCategories {
			fmt.Printf("[%s]\n", cat.category)
			for _, name := range cat.tools {
				fmt.Printf("  - %s\n", name)
				total++
			}
		}
		fmt.Printf("\n총 %d개 도구\n", total)
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}
