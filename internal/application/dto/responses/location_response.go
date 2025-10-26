package responses

type GHNAPIResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []T    `json:"data"`
}

type GeneralLocationResponse struct {
	NameExtension   []string `json:"NameExtension"`
	IsEnable        int      `json:"IsEnable"`
	CanUpdateCOD    bool     `json:"CanUpdateCOD"`
	UpdatedBy       int      `json:"UpdatedBy"`
	CreatedAt       string   `json:"CreatedAt"`
	UpdatedAt       string   `json:"UpdatedAt"`
	Status          int      `json:"Status"`
	UpdatedEmployee int      `json:"UpdatedEmployee"`
	UpdatedSource   string   `json:"UpdatedSource"`
	UpdatedDate     string   `json:"UpdatedDate"`
}

type ProvinceResponse struct {
	ProvinceID   int    `json:"ProvinceID"`
	ProvinceName string `json:"ProvinceName"`
	CountryID    int    `json:"CountryID"`
	Code         string `json:"Code"`
	RegionID     int    `json:"RegionID"`
	RegionCPN    int    `json:"RegionCPN"`
	GeneralLocationResponse
}

type DistrictResponse struct {
	DistrictID        int         `json:"DistrictID"`
	ProvinceID        int         `json:"ProvinceID"`
	DistrictName      string      `json:"DistrictName"`
	Code              string      `json:"Code"`
	Type              int         `json:"Type"`
	SupportType       int         `json:"SupportType"`
	PickType          int         `json:"PickType"`
	DeliverType       int         `json:"DeliverType"`
	WhiteListClient   interface{} `json:"WhiteListClient"`
	WhiteListDistrict interface{} `json:"WhiteListDistrict"`
	GovernmentCode    string      `json:"GovernmentCode"`
	ReasonCode        string      `json:"ReasonCode"`
	ReasonMessage     string      `json:"ReasonMessage"`
	OnDates           interface{} `json:"OnDates"`
	GeneralLocationResponse
}

type WardResponse struct {
	WardCode        string      `json:"WardCode"`
	DistrictID      int         `json:"DistrictID"`
	WardName        string      `json:"WardName"`
	SupportType     int         `json:"SupportType"`
	PickType        int         `json:"PickType"`
	DeliverType     int         `json:"DeliverType"`
	WhiteListClient interface{} `json:"WhiteListClient"`
	WhiteListWard   interface{} `json:"WhiteListWard"`
	GovernmentCode  string      `json:"GovernmentCode"`
	Config          interface{} `json:"Config"`
	ReasonCode      string      `json:"ReasonCode"`
	ReasonMessage   string      `json:"ReasonMessage"`
	OnDates         interface{} `json:"OnDates"`
	GeneralLocationResponse
}
