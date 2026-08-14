package tools

type kakaoResponse[T any] struct {
	Meta      kakaoMeta `json:"meta"`
	Documents []T       `json:"documents"`
}

type kakaoWebDocument struct {
	Title    string `json:"title"`
	Contents string `json:"contents"`
	URL      string `json:"url"`
	DateTime string `json:"datetime"`
}
type kakaoBlogDocument struct {
	Title     string `json:"title"`
	Contents  string `json:"contents"`
	URL       string `json:"url"`
	BlogName  string `json:"blogname"`
	Thumbnail string `json:"thumbnail"`
	DateTime  string `json:"datetime"`
}
type kakaoCafeDocument struct {
	Title     string `json:"title"`
	Contents  string `json:"contents"`
	URL       string `json:"url"`
	CafeName  string `json:"cafename"`
	Thumbnail string `json:"thumbnail"`
	DateTime  string `json:"datetime"`
}
type kakaoBookDocument struct {
	Title     string   `json:"title"`
	Contents  string   `json:"contents"`
	URL       string   `json:"url"`
	ISBN      string   `json:"isbn"`
	Authors   []string `json:"authors"`
	Publisher string   `json:"publisher"`
	DateTime  string   `json:"datetime"`
}
type kakaoImageDocument struct {
	ImageURL        string `json:"image_url"`
	ThumbnailURL    string `json:"thumbnail_url"`
	DisplaySiteName string `json:"display_sitename"`
	DocURL          string `json:"doc_url"`
	DateTime        string `json:"datetime"`
}
type kakaoVideoDocument struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Author    string `json:"author"`
	Thumbnail string `json:"thumbnail"`
	DateTime  string `json:"datetime"`
	PlayTime  int    `json:"play_time"`
}
type kakaoPlaceDocument struct {
	ID              string `json:"id"`
	PlaceName       string `json:"place_name"`
	CategoryName    string `json:"category_name"`
	Phone           string `json:"phone"`
	AddressName     string `json:"address_name"`
	RoadAddressName string `json:"road_address_name"`
	X               string `json:"x"`
	Y               string `json:"y"`
	PlaceURL        string `json:"place_url"`
	Distance        string `json:"distance"`
}
