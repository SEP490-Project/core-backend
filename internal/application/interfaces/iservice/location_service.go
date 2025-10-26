package iservice

import "core-backend/internal/application/dto/responses"

type LocationService interface {

	// Location Service
	GetProvinces() ([]responses.ProvinceResponse, error)
	GetDistrictsByProvinceID(provinceID int) ([]responses.DistrictResponse, error)
	GetWardsByDistrictID(districtID int) ([]responses.WardResponse, error)

	// Delivery Services

}
