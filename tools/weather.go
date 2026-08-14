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
	"github.com/wkqco33/public_api_mcp/cache"
	"github.com/wkqco33/public_api_mcp/config"
	"github.com/wkqco33/public_api_mcp/httpclient"
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

type WeatherClock interface{ Now() time.Time }
type WeatherHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}
type WeatherClient struct {
	httpClient WeatherHTTPDoer
	baseURL    string
	apiKey     string
}

func NewWeatherClient(client WeatherHTTPDoer, baseURL, apiKey string) *WeatherClient {
	return &WeatherClient{httpClient: client, baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey}
}
func defaultWeatherClient() *WeatherClient {
	return NewWeatherClient(httpclient.Default, "https://apis.data.go.kr/1360000/VilageFcstInfoService_2.0", config.KMAWeatherAPIKey)
}

type systemWeatherClock struct{}

func (systemWeatherClock) Now() time.Time { return time.Now() }

func RegisterWeather(s *mcp.Server) {
	RegisterWeatherWithDependencies(s, defaultWeatherClient(), systemWeatherClock{})
}
func RegisterWeatherWithClock(s *mcp.Server, clock WeatherClock) {
	RegisterWeatherWithDependencies(s, defaultWeatherClient(), clock)
}
func RegisterWeatherWithDependencies(s *mcp.Server, client *WeatherClient, clock WeatherClock) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_current_weather",
		Description: "기상청 초단기실황으로 현재 날씨(기온, 습도, 강수, 풍속) 조회.",
	}, withRecover("get_current_weather", func(ctx context.Context, req *mcp.CallToolRequest, in weatherInput) (*mcp.CallToolResult, any, error) {
		return getCurrentWeatherWithDependencies(ctx, client, clock, in)
	}))
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_weather_forecast",
		Description: "기상청 단기예보로 시간대별 날씨 예보 조회 (1~3일).",
	}, withRecover("get_weather_forecast", func(ctx context.Context, req *mcp.CallToolRequest, in weatherForecastInput) (*mcp.CallToolResult, any, error) {
		return getWeatherForecastWithDependencies(ctx, client, clock, in)
	}))
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

func kmaBaseTimeNcst() (string, string) { return kmaBaseTimeNcstAt(time.Now()) }

func kmaBaseTimeNcstAt(current time.Time) (string, string) {
	kst := time.FixedZone("KST", 9*60*60)
	now := current.In(kst)
	if now.Minute() < 10 {
		now = now.Add(-time.Hour)
	}
	return now.Format("20060102"), now.Format("15") + "00"
}

func kmaBaseTimeVilage() (string, string) { return kmaBaseTimeVilageAt(time.Now()) }

func kmaBaseTimeVilageAt(current time.Time) (string, string) {
	kst := time.FixedZone("KST", 9*60*60)
	now := current.In(kst)
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

type kmaItem struct {
	Category  string `json:"category"`
	ObsrValue string `json:"obsrValue"`
	FcstDate  string `json:"fcstDate"`
	FcstTime  string `json:"fcstTime"`
	FcstValue string `json:"fcstValue"`
}
type kmaBody struct {
	Items struct {
		Item []kmaItem `json:"item"`
	} `json:"items"`
}
type kmaAPIResponse struct {
	Response struct {
		Header struct {
			ResultCode string `json:"resultCode"`
			ResultMsg  string `json:"resultMsg"`
		} `json:"header"`
		Body kmaBody `json:"body"`
	} `json:"response"`
}

func kmaGet(ctx context.Context, endpoint string, params map[string]string) (kmaBody, error) {
	return kmaGetWithClient(ctx, defaultWeatherClient(), endpoint, params)
}
func kmaGetWithClient(ctx context.Context, client *WeatherClient, endpoint string, params map[string]string) (kmaBody, error) {
	if client.apiKey == "" {
		return kmaBody{}, fmt.Errorf("KMA_WEATHER_API_KEY가 설정되지 않았습니다")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", client.baseURL+"/"+endpoint, nil)
	if err != nil {
		return kmaBody{}, fmt.Errorf("요청 생성 실패: %w", err)
	}
	q := req.URL.Query()
	q.Set("serviceKey", client.apiKey)
	q.Set("dataType", "JSON")
	q.Set("pageNo", "1")
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	body, _, err := httpclient.FetchBodyWithClient(client.httpClient, req, maxAPIBodyBytes)
	if err != nil {
		return kmaBody{}, err
	}

	var data kmaAPIResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return kmaBody{}, fmt.Errorf("응답 파싱 실패: %v", err)
	}
	if code := data.Response.Header.ResultCode; code != "00" && code != "0" {
		return kmaBody{}, fmt.Errorf("KMA API 오류: %s", data.Response.Header.ResultMsg)
	}
	return data.Response.Body, nil
}

func getCurrentWeather(ctx context.Context, _ *mcp.CallToolRequest, in weatherInput) (*mcp.CallToolResult, any, error) {
	return getCurrentWeatherWithClock(ctx, systemWeatherClock{}, in)
}

func getCurrentWeatherWithClock(ctx context.Context, clock WeatherClock, in weatherInput) (*mcp.CallToolResult, any, error) {
	return getCurrentWeatherWithDependencies(ctx, defaultWeatherClient(), clock, in)
}
func getCurrentWeatherWithDependencies(ctx context.Context, client *WeatherClient, clock WeatherClock, in weatherInput) (*mcp.CallToolResult, any, error) {
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

	baseDate, baseTime := kmaBaseTimeNcstAt(clock.Now())
	body, err := kmaGetWithClient(ctx, client, "getUltraSrtNcst", map[string]string{
		"numOfRows": "10",
		"base_date": baseDate,
		"base_time": baseTime,
		"nx":        loc.NX,
		"ny":        loc.NY,
	})
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	vals := map[string]string{}
	for _, item := range body.Items.Item {
		vals[item.Category] = item.ObsrValue
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

func getWeatherForecast(ctx context.Context, _ *mcp.CallToolRequest, in weatherForecastInput) (*mcp.CallToolResult, any, error) {
	return getWeatherForecastWithClock(ctx, systemWeatherClock{}, in)
}

func getWeatherForecastWithClock(ctx context.Context, clock WeatherClock, in weatherForecastInput) (*mcp.CallToolResult, any, error) {
	return getWeatherForecastWithDependencies(ctx, defaultWeatherClient(), clock, in)
}
func getWeatherForecastWithDependencies(ctx context.Context, client *WeatherClient, clock WeatherClock, in weatherForecastInput) (*mcp.CallToolResult, any, error) {
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

	baseDate, baseTime := kmaBaseTimeVilageAt(clock.Now())
	numRows := days * 24 * 12
	body, err := kmaGetWithClient(ctx, client, "getVilageFcst", map[string]string{
		"numOfRows": fmt.Sprintf("%d", numRows),
		"base_date": baseDate,
		"base_time": baseTime,
		"nx":        loc.NX,
		"ny":        loc.NY,
	})
	if err != nil {
		return toolError("%v", err), nil, nil
	}

	type slot struct {
		date string
		time string
		vals map[string]string
	}
	slots := map[string]*slot{}
	var slotOrder []string
	for _, item := range body.Items.Item {
		fDate, fTime, cat, val := item.FcstDate, item.FcstTime, item.Category, item.FcstValue
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
