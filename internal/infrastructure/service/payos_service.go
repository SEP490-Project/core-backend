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
	"time"

	"go.uber.org/zap"
)

type payOsService struct {
	PaymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
	config                       *config.AppConfig
}

func (p payOsService) UpdatePaymentStatus(orderID int64, status string, reason string) error {
	//TODO implement me
	zap.L().Info("UPDATE PAYMENT STATUS", zap.Int("orderId", int(orderID)), zap.String("status", status), zap.String("reason", reason))
	// 1. Tìm payment transaction theo orderId
	// 2. Cập nhật trạng thái payment transaction
	// 3. Nếu trạng thái là thành công, cập nhật đơn hàng tương ứng
	// 4. Xử lý các trạng thái khác nếu cần thiết
	return nil
}

func (p payOsService) GetPayOSOrderInfo(orderId string) (*responses.PayOSWrapperResponse[responses.PayOSOrderInfoResponse], error) {
	url := p.config.PayOS.BaseURL
	secondTimeout := p.config.Server.Timeout

	httpReq, err := http.NewRequest("GET", url+"/"+orderId, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-client-id", p.config.PayOS.ClientID)
	httpReq.Header.Set("x-api-key", p.config.PayOS.APIKey)

	client := &http.Client{Timeout: time.Duration(secondTimeout) * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // <-- thêm dòng này

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
	// 1) setup
	orderCode := generateOrderCode()
	cancelUrl := p.config.PayOS.CancelURL
	returnUrl := p.config.PayOS.ReturnURL

	if len([]rune(req.Description)) > 9 {
		req.Description = string([]rune(req.Description)[:9])
	}

	sig, err := p.generateSignature(
		req.Amount,
		cancelUrl,
		req.Description,
		orderCode,
		returnUrl,
	)
	if err != nil {
		return nil, err
	}

	// 2) Build payload
	payload := p.buildPayOSRequest(req, orderCode, cancelUrl, returnUrl, sig)
	body, _ := json.Marshal(payload)

	// 3) Call PayOS
	httpReq, err := http.NewRequest("POST", p.config.PayOS.BaseURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-client-id", p.config.PayOS.ClientID)
	httpReq.Header.Set("x-api-key", p.config.PayOS.APIKey)

	client := &http.Client{Timeout: time.Duration(p.config.Server.Timeout) * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Printf("PAYOS response: %s\n", string(bodyBytes))

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
// generateSignature tạo HMAC-SHA256 signature cho PayOS theo đúng thứ tự trường.
func (p payOsService) generateSignature(amount int64, cancelUrl, description string, orderCode int64, returnUrl string) (string, error) {
	data := fmt.Sprintf(
		"amount=%d&cancelUrl=%s&description=%s&orderCode=%d&returnUrl=%s",
		amount, cancelUrl, description, orderCode, returnUrl,
	)
	mac := hmac.New(sha256.New, []byte(p.config.PayOS.ChecksumKey))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (p payOsService) buildPayOSRequest(
	req requests.PaymentRequest,
	orderCode int64,
	cancelURL, returnURL, signature string,
) requests.PayOSRequest {
	expiredAt := time.Now().Add(time.Duration(p.config.Server.PayOSLinkExpiry) * time.Second).Unix()

	items := make([]responses.PaymentItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, responses.PaymentItem{
			Name:     it.Name,
			Quantity: it.Quantity,
			Price:    float64(it.Price),
		})
	}

	var inv responses.Invoice
	if req.Invoice != nil {
		inv = responses.Invoice{
			BuyerNotGetInvoice: false,
			TaxPercentage:      0,
		}
	}

	return requests.PayOSRequest{
		PaymentSignatureRequest: requests.PaymentSignatureRequest{
			Amount:      req.Amount,
			CancelUrl:   cancelURL,
			Description: req.Description,
			OrderCode:   orderCode,
			ReturnUrl:   returnURL,
		},
		BuyerName:        utils.StrPtrOrNil(req.BuyerName),
		BuyerCompanyName: utils.StrPtrOrNil(req.BuyerCompanyName),
		BuyerTaxCode:     utils.StrPtrOrNil(req.BuyerTaxCode),
		BuyerAddress:     utils.StrPtrOrNil(req.BuyerAddress),
		BuyerEmail:       utils.StrPtrOrNil(req.BuyerEmail),
		BuyerPhone:       utils.StrPtrOrNil(req.BuyerPhone),
		Items:            items,
		Invoice:          inv,
		ExpiredAt:        expiredAt,
		Signature:        signature,
	}
}

func generateOrderCode() int64 {
	now := time.Now().Unix()
	randPart := time.Now().UnixNano() % 1e3
	return now*1000 + randPart
}
