package tools

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"wkqcosoft.com/public-api-mcp/cache"
	"wkqcosoft.com/public-api-mcp/config"
	"wkqcosoft.com/public-api-mcp/httpclient"
)

//go:embed kma_weather_location.csv
var kmaLocationCSV []byte

type kmaLocation struct {
	Step1 string
	Step2 string
	Step3 string
	NX    string
	NY    string
}

var (
	kmaLocations []kmaLocation
	kmaOnce      sync.Once
)

var weatherCache = cache.New[string, string]()
var locationCache = cache.New[string, *kmaLocation]()

var ptyMap = map[string]string{
	"0": "없음", "1": "비", "2": "비/눈", "3": "눈",
	"5": "빗방울", "6": "빗방울눈날림", "7": "눈날림",
}
var skyMap = map[string]string{"1": "맑음", "3": "구름많음", "4": "흐림"}

type weatherInput struct {
	Location string `json:"location" jsonschema:"지역명 (예: 서울, 강남구, 역삼동)"`
}

type weatherForecastInput struct {
	Location string `json:"location" jsonschema:"지역명 (예: 서울, 강남구, 역삼동)"`
	Days     int    `json:"days,omitempty" jsonschema:"예보 일수 1~3 (기본: 1)"`
}

func RegisterWeather(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_current_weather",
		Description: "기상청 초단기실황으로 현재 날씨(기온, 습도, 강수, 풍속) 조회.",
	}, getCurrentWeather)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_weather_forecast",
		Description: "기상청 단기예보로 시간대별 날씨 예보 조회 (1~3일).",
	}, getWeatherForecast)
}

func loadKMALocations() {
	kmaOnce.Do(func() {
		r := csv.NewReader(bytes.NewReader(kmaLocationCSV))
		r.LazyQuotes = true
		header, err := r.Read()
		if err != nil {
			return
		}
		idx := map[string]int{}
		for i, h := range header {
			idx[strings.TrimSpace(h)] = i
		}
		for {
			row, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			get := func(key string) string {
				i, ok := idx[key]
				if !ok || i >= len(row) {
					return ""
				}
				return strings.TrimSpace(row[i])
			}
			kmaLocations = append(kmaLocations, kmaLocation{
				Step1: get("1단계"),
				Step2: get("2단계"),
				Step3: get("3단계"),
				NX:    get("격자 X"),
				NY:    get("격자 Y"),
			})
		}
		slog.Info("기상청 지역 좌표 CSV 로드 완료", "count", len(kmaLocations))
	})
}

func findLocation(name string) *kmaLocation {
	loadKMALocations()
	name = strings.ReplaceAll(name, "날씨", "")
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if cached, ok := locationCache.Get(name); ok {
		return cached
	}
	// 3단계 → 2단계 → 1단계 순서로 검색
	var found *kmaLocation
	for _, step := range []int{3, 2, 1} {
		for i := range kmaLocations {
			loc := &kmaLocations[i]
			var field string
			switch step {
			case 3:
				field = loc.Step3
			case 2:
				field = loc.Step2
			case 1:
				field = loc.Step1
			}
			if strings.Contains(field, name) {
				found = loc
				break
			}
		}
		if found != nil {
			break
		}
	}
	locationCache.Set(name, found, 24*time.Hour)
	return found
}

func kmaBaseTimeNcst() (string, string) {
	kst := time.FixedZone("KST", 9*60*60)
	now := time.Now().In(kst)
	if now.Minute() < 10 {
		now = now.Add(-time.Hour)
	}
	return now.Format("20060102"), now.Format("15") + "00"
}

func kmaBaseTimeVilage() (string, string) {
	kst := time.FixedZone("KST", 9*60*60)
	now := time.Now().In(kst)
	times := []int{2, 5, 8, 11, 14, 17, 20, 23}
	hour := now.Hour()
	if now.Minute() < 10 {
		hour--
	}
	best := -1
	for _, t := range times {
		if t <= hour {
			best = t
		}
	}
	if best < 0 {
		now = now.AddDate(0, 0, -1)
		return now.Format("20060102"), "2300"
	}
	return now.Format("20060102"), fmt.Sprintf("%02d00", best)
}

func kmaGet(endpoint string, params map[string]string) (map[string]any, error) {
	if config.KMAWeatherAPIKey == "" {
		return nil, fmt.Errorf("KMA_WEATHER_API_KEY가 설정되지 않았습니다")
	}
	req, _ := http.NewRequest("GET", "http://apis.data.go.kr/1360000/VilageFcstInfoService_2.0/"+endpoint, nil)
	q := req.URL.Query()
	q.Set("serviceKey", config.KMAWeatherAPIKey)
	q.Set("dataType", "JSON")
	q.Set("pageNo", "1")
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	res, err := httpclient.Default.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 요청 실패: %v", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %v", err)
	}
	response, _ := data["response"].(map[string]any)
	header, _ := response["header"].(map[string]any)
	if code, _ := header["resultCode"].(string); code != "00" && code != "0" {
		msg, _ := header["resultMsg"].(string)
		return nil, fmt.Errorf("KMA API 오류: %s", msg)
	}
	respBody, _ := response["body"].(map[string]any)
	return respBody, nil
}

func getCurrentWeather(_ context.Context, _ *mcp.CallToolRequest, in weatherInput) (*mcp.CallToolResult, any, error) {
	loc := findLocation(in.Location)
	if loc == nil {
		return toolError("'%s' 지역을 찾을 수 없습니다", in.Location), nil, nil
	}

	cacheKey := "ncst:" + in.Location
	if cached, ok := weatherCache.Get(cacheKey); ok {
		slog.Debug("날씨 캐시 hit", "key", cacheKey)
		return toolText(cached), nil, nil
	}
	slog.Debug("날씨 API 조회", "location", in.Location, "nx", loc.NX, "ny", loc.NY)

	baseDate, baseTime := kmaBaseTimeNcst()
	body, err := kmaGet("getUltraSrtNcst", map[string]string{
		"numOfRows": "10",
		"base_date": baseDate,
		"base_time": baseTime,
		"nx":        loc.NX,
		"ny":        loc.NY,
	})
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	items_, _ := body["items"].(map[string]any)
	itemList, _ := items_["item"].([]any)
	vals := map[string]string{}
	for _, raw := range itemList {
		item, _ := raw.(map[string]any)
		cat, _ := item["category"].(string)
		val, _ := item["obsrValue"].(string)
		vals[cat] = val
	}

	region := strings.TrimSpace(loc.Step1 + " " + loc.Step2 + " " + loc.Step3)
	temp := vals["T1H"]
	hum := vals["REH"]
	pty := ptyMap[vals["PTY"]]
	if pty == "" {
		pty = "없음"
	}
	wsd := vals["WSD"]
	rn1 := vals["RN1"]
	rainStr := ""
	if rn1 != "" && rn1 != "0" && rn1 != "강수없음" {
		rainStr = fmt.Sprintf(" / 강수 %smm", rn1)
	}

	result := fmt.Sprintf("📍 %s: %s°C / 습도 %s%% / %s / 풍속 %sm/s%s", region, temp, hum, pty, wsd, rainStr)
	weatherCache.Set(cacheKey, result, 10*time.Minute)
	return toolText(result), nil, nil
}

func getWeatherForecast(_ context.Context, _ *mcp.CallToolRequest, in weatherForecastInput) (*mcp.CallToolResult, any, error) {
	loc := findLocation(in.Location)
	if loc == nil {
		return toolError("'%s' 지역을 찾을 수 없습니다", in.Location), nil, nil
	}

	days := in.Days
	if days <= 0 {
		days = 1
	}
	if days > 3 {
		days = 3
	}

	cacheKey := fmt.Sprintf("vilage:%s:%d", in.Location, days)
	if cached, ok := weatherCache.Get(cacheKey); ok {
		slog.Debug("예보 캐시 hit", "key", cacheKey)
		return toolText(cached), nil, nil
	}
	slog.Debug("예보 API 조회", "location", in.Location, "days", days)

	baseDate, baseTime := kmaBaseTimeVilage()
	numRows := days * 24 * 12
	body, err := kmaGet("getVilageFcst", map[string]string{
		"numOfRows": fmt.Sprintf("%d", numRows),
		"base_date": baseDate,
		"base_time": baseTime,
		"nx":        loc.NX,
		"ny":        loc.NY,
	})
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	items_, _ := body["items"].(map[string]any)
	itemList, _ := items_["item"].([]any)

	type slot struct {
		date string
		time string
		vals map[string]string
	}
	slots := map[string]*slot{}
	var slotOrder []string
	for _, raw := range itemList {
		item, _ := raw.(map[string]any)
		fDate, _ := item["fcstDate"].(string)
		fTime, _ := item["fcstTime"].(string)
		cat, _ := item["category"].(string)
		val, _ := item["fcstValue"].(string)
		key := fDate + fTime
		if _, ok := slots[key]; !ok {
			slots[key] = &slot{date: fDate, time: fTime, vals: map[string]string{}}
			slotOrder = append(slotOrder, key)
		}
		slots[key].vals[cat] = val
	}

	kst := time.FixedZone("KST", 9*60*60)
	now := time.Now().In(kst)
	region := strings.TrimSpace(loc.Step1 + " " + loc.Step2 + " " + loc.Step3)
	var sb strings.Builder
	fmt.Fprintf(&sb, "📍 %s 날씨 예보 (%d일)\n", region, days)

	count := 0
	for _, key := range slotOrder {
		s := slots[key]
		// 3시간 간격만 표시
		h := 0
		fmt.Sscanf(s.time[:2], "%d", &h)
		if h%3 != 0 {
			continue
		}
		t, err := time.ParseInLocation("20060102 1504", s.date+" "+s.time, kst)
		if err != nil {
			continue
		}
		if t.Before(now) || t.After(now.AddDate(0, 0, days)) {
			continue
		}

		sky := skyMap[s.vals["SKY"]]
		if sky == "" {
			sky = "-"
		}
		pty := ptyMap[s.vals["PTY"]]
		if pty == "" {
			pty = "없음"
		}
		tmp := s.vals["TMP"]
		pop := s.vals["POP"]
		wsd := s.vals["WSD"]

		fmt.Fprintf(&sb, "%s %s | %s°C | %s/%s | 강수확률 %s%% | 풍속 %sm/s\n",
			t.Format("01/02"), t.Format("15:04"), tmp, sky, pty, pop, wsd)
		count++
	}
	if count == 0 {
		sb.WriteString("예보 데이터가 없습니다\n")
	}

	result := sb.String()
	weatherCache.Set(cacheKey, result, 30*time.Minute)
	return toolText(result), nil, nil
}
