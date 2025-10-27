package service

import (
	"bytes"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type locationService struct {
}

func (l locationService) InputUserAddress(userID string, addressReq requests.InputAddressRequest) (*model.ShippingAddress, error) {
	//TODO implement me
	panic("implement me")
}

func (l locationService) SetAddressAsDefault(userID string, addressID string) error {
	//TODO implement me
	panic("implement me")
}

func (l locationService) GetProvinces() ([]responses.ProvinceResponse, error) {
	cfg := config.GetAppConfig()
	url := cfg.GHN.BaseURL + "/province"
	token := cfg.GHN.Token
	return doRequest[responses.ProvinceResponse]("GET", url, token, nil)
}

func (l locationService) GetDistrictsByProvinceID(provinceID int) ([]responses.DistrictResponse, error) {
	cfg := config.GetAppConfig()
	url := fmt.Sprintf("%s/district?province_id=%d", cfg.GHN.BaseURL, provinceID)
	token := cfg.GHN.Token
	return doRequest[responses.DistrictResponse]("GET", url, token, nil)
}

func (l locationService) GetWardsByDistrictID(districtID int) ([]responses.WardResponse, error) {
	loadedCfg := config.GetAppConfig()
	url := fmt.Sprintf("%s/ward?district_id=%d", loadedCfg.GHN.BaseURL, districtID)
	token := loadedCfg.GHN.Token
	return doRequest[responses.WardResponse]("GET", url, token, nil)
}

func NewLocationService() iservice.LocationService {
	return &locationService{}
}

func doRequest[T any](method, url, token string, body any) ([]T, error) {
	var buf io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to marshal body: %s", err))
			return nil, err
		}
		buf = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to create request: %s", err))
		return nil, err
	}
	req.Header.Add("Token", token)
	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to send request: %s", err))
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to read response: %s", err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		zap.L().Warn(fmt.Sprintf("GHN API returned non-200: %s\nBody: %s", resp.Status, string(respBody)))
	}

	var result responses.GHNAPIResponse[T]
	if err := json.Unmarshal(respBody, &result); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to unmarshal response: %s", err))
		return nil, err
	}
	return result.Data, nil
}
