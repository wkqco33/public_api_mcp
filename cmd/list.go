package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var toolCategories = []struct {
	category string
	tools    []string
}{
	{"kakao (Daum 검색 + 카카오맵 Local API)", []string{
		"kakao_search_web", "kakao_search_blog", "kakao_search_cafe",
		"kakao_search_book", "kakao_search_image", "kakao_search_video",
		"kakao_search_places",
	}},
	{"exchange (한국수출입은행)", []string{
		"get_exchange_rates",
	}},
	{"fuel (오피넷)", []string{
		"get_fuel_avg_price", "get_fuel_sido_price", "get_fuel_low_top10",
	}},
	{"stock (공공데이터포털)", []string{
		"get_stock_price", "get_stock_analysis_data", "get_stock_market_index",
	}},
	{"real_estate (공공데이터포털)", []string{
		"get_real_estate_data",
	}},
	{"transport (공공데이터포털 + 서울시)", []string{
		"search_bus_stop", "get_bus_arrival", "get_subway_arrival",
	}},
	{"weather (기상청)", []string{
		"get_current_weather", "get_weather_forecast",
	}},
	{"dart (금융감독원 DART)", []string{
		"search_dart_company", "get_dart_disclosures", "get_dart_financials",
		"get_dart_dividend_info", "get_dart_multi_financials",
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
