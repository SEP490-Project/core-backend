package iservice

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type LocationService interface {

	// Location Service
	GetProvinces() ([]responses.ProvinceResponse, error)
	GetDistrictsByProvinceID(provinceID int) ([]responses.DistrictResponse, error)
	GetWardsByDistrictID(districtID int) ([]responses.WardResponse, error)

	// Delivery Services
	InputUserAddress(userID uuid.UUID, addressReq requests.InputAddressRequest) (*model.ShippingAddress, error)
	SetAddressAsDefault(userID string, addressID string) error
	GetUserAddresses(userID uuid.UUID) ([]model.ShippingAddress, error)
}
