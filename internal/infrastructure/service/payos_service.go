package service

import (
	"bytes"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type payOsService struct {
	PaymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
	config                       *config.AppConfig
}

func (p payOsService) GetPayOSOrderInfo(orderId string) (*responses.PayOSWrapperResponse[responses.PayOSOrderInfoResponse], error) {
	// get value form config
	url := p.config.PayOS.BaseUrl
	secondTimeout := p.config.Server.Timeout

	//build payos request
	httpReq, err := http.NewRequest("GET", url+"/"+orderId, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-client-id", p.config.PayOS.ClientID)
	httpReq.Header.Set("x-api-key", p.config.PayOS.ApiKey)

	client := &http.Client{Timeout: time.Duration(secondTimeout) * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var payRes responses.PayOSWrapperResponse[responses.PayOSOrderInfoResponse]
	if err := json.NewDecoder(resp.Body).Decode(&payRes); err != nil {
		return nil, err
	}

	return &payRes, nil
}

func (p payOsService) CancelPayOSLink(paymentId string, cancellationReason string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (p payOsService) GeneratePayOSLink(req requests.PaymentRequest) (*responses.PayOSWrapperResponse[responses.PayOSLinkResponse], error) {
	// get value form config
	url := p.config.PayOS.BaseUrl
	secondTimeout := p.config.Server.Timeout

	//Generate orderCode
	orderCode := generateOrderCode()
	oc := int(orderCode)
	// ensure request carries the generated order code
	req.OrderCode = &oc

	// sign request
	signReq := requests.PaymentSignatureRequest{
		Amount:      req.Amount,
		CancelUrl:   req.CancelUrl,
		Description: req.Description,
		OrderCode:   &oc,
		ReturnUrl:   req.ReturnUrl,
	}

	sig, err := p.generateSignature(signReq)
	if err != nil {
		return nil, err
	}

	// build payos request
	payload := p.buildPayOSRequest(req, sig)
	body, _ := json.Marshal(payload)

	// call payos api
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))

	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-client-id", p.config.PayOS.ClientID)
	httpReq.Header.Set("x-api-key", p.config.PayOS.ApiKey)

	client := &http.Client{Timeout: time.Duration(secondTimeout) * time.Second}
	resp, err := client.Do(httpReq)

	//log:

	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Printf("PAYOS response: %s\n", string(bodyBytes))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var payRes responses.PayOSWrapperResponse[responses.PayOSLinkResponse]
	if err := json.Unmarshal(bodyBytes, &payRes); err != nil {
		return nil, err
	}

	return &payRes, nil
}

func NewPayOsService(paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]) iservice_third_party.PayOSService {
	return &payOsService{
		PaymentTransactionRepository: paymentTransactionRepository,
		config:                       config.GetAppConfig(),
	}
}

// PRIVATE
func (p payOsService) generateSignature(req requests.PaymentSignatureRequest) (string, error) {
	values := make(map[string]interface{})

	// Add fields unconditionally that are always required
	values["amount"] = req.Amount
	values["cancelUrl"] = req.CancelUrl
	values["description"] = req.Description
	if req.OrderCode != nil { // only include when present, and use its value
		values["orderCode"] = *req.OrderCode
	}
	values["returnUrl"] = req.ReturnUrl

	// Step 2: Sort the keys alphabetically.
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Step 3: Build the canonical data string (key1=value1&key2=value2...).
	var sb strings.Builder
	for i, k := range keys {
		valAsString := fmt.Sprintf("%v", values[k])
		sb.WriteString(fmt.Sprintf("%s=%s", k, valAsString))
		if i < len(keys)-1 {
			sb.WriteString("&")
		}
	}

	dataToSign := sb.String()
	fmt.Printf("checksum key: %s\n", p.config.PayOS.ChecksumKey)
	fmt.Printf("Data to sign: %s\n", dataToSign) // This will now be different if fields were empty

	// Step 4 & 5: Compute HMAC and encode
	h := hmac.New(sha256.New, []byte(p.config.PayOS.ChecksumKey))
	h.Write([]byte(dataToSign))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (p payOsService) buildPayOSRequest(req requests.PaymentRequest, signature string) requests.PayOSRequest {

	//custom fields
	expiredTime := p.config.Server.PayOSLinkExpiry
	expiredAt := time.Now().Add(time.Duration(expiredTime) * time.Second).Unix()

	// build request
	return requests.PayOSRequest{
		PaymentSignatureRequest: requests.PaymentSignatureRequest{
			Amount:      req.Amount,
			CancelUrl:   req.CancelUrl,
			Description: req.Description,
			OrderCode:   req.OrderCode,
			ReturnUrl:   req.ReturnUrl,
		},
		BuyerName:        utils.StrPtrOrNil(req.BuyerName),
		BuyerCompanyName: utils.StrPtrOrNil(req.BuyerCompanyName),
		BuyerTaxCode:     utils.StrPtrOrNil(req.BuyerTaxCode),
		BuyerAddress:     utils.StrPtrOrNil(req.BuyerAddress),
		BuyerEmail:       utils.StrPtrOrNil(req.BuyerEmail),
		BuyerPhone:       utils.StrPtrOrNil(req.BuyerPhone),
		Items:            []responses.PaymentItem{},
		Invoice:          responses.Invoice{},
		ExpiredAt:        expiredAt,
		Signature:        signature,
	}
}

func generateOrderCode() int64 {
	return time.Now().UnixNano()
}
