package proxies

import (
	"bytes"
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"go.uber.org/zap"
)

type payosProxy struct {
	*BaseProxy
	clientID    string
	apiKey      string
	checksumKey string
}

// CreatePaymentLink implements iproxies.PayOSProxy
func (p *payosProxy) CreatePaymentLink(ctx context.Context, req *dtos.PayOSCreateLinkRequest) (*responses.PayOSLinkResponse, error) {
	// Marshal request body
	body, err := json.Marshal(req)
	if err != nil {
		zap.L().Error("Failed to marshal PayOS create link request", zap.Error(err))
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v2/payment-requests", bytes.NewBuffer(body))
	if err != nil {
		zap.L().Error("Failed to create HTTP request", zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-client-id", p.clientID)
	httpReq.Header.Set("x-api-key", p.apiKey)

	// Execute request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		zap.L().Error("Failed to execute PayOS create link request", zap.Error(err))
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var payosResp dtos.PayOSWrapperResponse[responses.PayOSLinkResponse]
	if err := json.NewDecoder(resp.Body).Decode(&payosResp); err != nil {
		zap.L().Error("Failed to decode PayOS response", zap.Error(err), zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check response code
	if payosResp.Code != "00" {
		zap.L().Error("PayOS returned error",
			zap.String("code", payosResp.Code),
			zap.String("desc", payosResp.Desc),
			zap.Int("http_status", resp.StatusCode))
		return nil, fmt.Errorf("PayOS error: %s - %s", payosResp.Code, payosResp.Desc)
	}

	zap.L().Info("PayOS payment link created successfully",
		zap.String("payment_link_id", payosResp.Data.PaymentLinkID),
		zap.Int64("order_code", int64(payosResp.Data.OrderCode)))

	return &payosResp.Data, nil
}

// GetPaymentInfo implements iproxies.PayOSProxy
func (p *payosProxy) GetPaymentInfo(ctx context.Context, orderCode string) (*responses.PayOSOrderInfoResponse, error) {
	// Create HTTP request
	url := fmt.Sprintf("%s/v2/payment-requests/%s", p.baseURL, orderCode)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		zap.L().Error("Failed to create HTTP request", zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-client-id", p.clientID)
	httpReq.Header.Set("x-api-key", p.apiKey)

	// Execute request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		zap.L().Error("Failed to execute PayOS get payment info request", zap.Error(err))
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var payosResp dtos.PayOSWrapperResponse[responses.PayOSOrderInfoResponse]
	if err := json.NewDecoder(resp.Body).Decode(&payosResp); err != nil {
		zap.L().Error("Failed to decode PayOS response", zap.Error(err), zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check response code
	if payosResp.Code != "00" {
		zap.L().Error("PayOS returned error",
			zap.String("code", payosResp.Code),
			zap.String("desc", payosResp.Desc),
			zap.Int("http_status", resp.StatusCode))
		return nil, fmt.Errorf("PayOS error: %s - %s", payosResp.Code, payosResp.Desc)
	}

	return &payosResp.Data, nil
}

// CancelPaymentLink implements iproxies.PayOSProxy
func (p *payosProxy) CancelPaymentLink(ctx context.Context, orderCode string, reason string) (*responses.PayOSOrderInfoResponse, error) {
	// Create request body
	cancelReq := dtos.PayOSCancelRequest{
		CancellationReason: reason,
	}
	body, err := json.Marshal(cancelReq)
	if err != nil {
		zap.L().Error("Failed to marshal PayOS cancel request", zap.Error(err))
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v2/payment-requests/%s/cancel", p.baseURL, orderCode)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		zap.L().Error("Failed to create HTTP request", zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-client-id", p.clientID)
	httpReq.Header.Set("x-api-key", p.apiKey)

	// Execute request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		zap.L().Error("Failed to execute PayOS cancel link request", zap.Error(err))
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var payosResp dtos.PayOSWrapperResponse[responses.PayOSOrderInfoResponse]
	if err := json.NewDecoder(resp.Body).Decode(&payosResp); err != nil {
		zap.L().Error("Failed to decode PayOS response", zap.Error(err), zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check response code
	if payosResp.Code == "221" {
		zap.L().Info("PayOS payment link already cancelled",
			zap.String("order_code", orderCode),
			zap.String("reason", reason))
		return &payosResp.Data, nil
	} else if payosResp.Code != "00" {
		zap.L().Error("PayOS returned error",
			zap.String("code", payosResp.Code),
			zap.String("desc", payosResp.Desc),
			zap.Int("http_status", resp.StatusCode))
		return &payosResp.Data, fmt.Errorf("PayOS error: %s - %s", payosResp.Code, payosResp.Desc)
	}

	zap.L().Info("PayOS payment link cancelled successfully",
		zap.String("order_code", orderCode),
		zap.String("reason", reason))

	return &payosResp.Data, nil
}

// ConfirmWebhookURL implements iproxies.PayOSProxy
func (p *payosProxy) ConfirmWebhookURL(ctx context.Context, webhookURL string) (*dtos.PayOSConfirmWebhookResponse, error) {
	var (
		request = dtos.PayOSConfirmWebhookRequest{WebhookURL: webhookURL}
		url     = "/confirm-webhook"
		headers = map[string]string{
			"Content-Type": "application/json",
			"x-client-id":  p.clientID,
			"x-api-key":    p.apiKey,
		}
	)
	resp, err := p.Post(ctx, url, headers, request)
	if err != nil {
		zap.L().Error("Failed to confirm PayOS webhook URL", zap.Error(err))
		return nil, fmt.Errorf("failed to confirm webhook URL: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var payosResp dtos.PayOSWrapperResponse[any]
	if err := json.NewDecoder(resp.Body).Decode(&payosResp); err != nil {
		zap.L().Error("Failed to decode PayOS response", zap.Error(err), zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	zap.L().Debug("PayOS confirm webhook response decoded", zap.Any("response", payosResp))
	if resp.StatusCode != http.StatusOK {
		zap.L().Error("PayOS returned error",
			zap.String("code", resp.Status),
			zap.String("desc", resp.Status),
			zap.Int("http_status", resp.StatusCode))
		return nil, fmt.Errorf("PayOS error: %s - %s", resp.Status, resp.Status)
	}

	// Check response code
	if payosResp.Code != "00" {
		zap.L().Error("PayOS returned error",
			zap.String("code", payosResp.Code),
			zap.String("desc", payosResp.Desc),
			zap.Int("http_status", resp.StatusCode))
		return nil, fmt.Errorf("PayOS error: %s - %s", payosResp.Code, payosResp.Desc)
	}

	var payosRespData dtos.PayOSConfirmWebhookResponse
	if rawData, err := json.Marshal(payosResp.Data); err != nil {
		zap.L().Error("Failed to marshal PayOS confirm webhook response",
			zap.Error(err))
		return nil, fmt.Errorf("failed to marshal PayOS confirm webhook response: %w", err)
	} else {
		if err = json.Unmarshal(rawData, &payosRespData); err != nil {
			zap.L().Error("Failed to unmarshal PayOS confirm webhook response",
				zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal PayOS confirm webhook response: %w", err)
		}
	}

	zap.L().Info("PayOS webhook URL confirmed successfully",
		zap.String("webhook_url", payosRespData.WebhookURL))

	return &payosRespData, nil
}

// VerifyWebhookSignature implements iproxies.PayOSProxy
func (p *payosProxy) VerifyWebhookSignature(data []byte, signature string) bool {
	// Parse JSON data into a map
	var dataMap map[string]any
	if err := json.Unmarshal(data, &dataMap); err != nil {
		zap.L().Error("Failed to parse webhook data for signature verification", zap.Error(err))
		return false
	}

	// Sort keys alphabetically
	keys := make([]string, 0, len(dataMap))
	for key := range dataMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build query string: key1=value1&key2=value2&...
	var queryParts []string
	for _, key := range keys {
		value := dataMap[key]
		// Convert value to string representation
		var valueStr string
		switch v := value.(type) {
		case string:
			valueStr = v
		case float64:
			// Handle numbers (JSON numbers are float64)
			if v == float64(int64(v)) {
				valueStr = fmt.Sprintf("%.0f", v) // No decimal for integers
			} else {
				valueStr = fmt.Sprintf("%v", v)
			}
		case bool:
			valueStr = fmt.Sprintf("%v", v)
		case nil:
			valueStr = ""
		default:
			// For nested objects/arrays, marshal to JSON
			bytes, _ := json.Marshal(v)
			valueStr = string(bytes)
		}
		queryParts = append(queryParts, fmt.Sprintf("%s=%s", key, valueStr))
	}
	queryString := strings.Join(queryParts, "&")

	// Create HMAC-SHA256 hash of the query string
	mac := hmac.New(sha256.New, []byte(p.checksumKey))
	mac.Write([]byte(queryString))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	expectedBytes, _ := hex.DecodeString(expectedSignature)
	isValid := hmac.Equal(sigBytes, expectedBytes)

	if !isValid {
		zap.L().Warn("PayOS webhook signature verification failed",
			zap.String("query_string", queryString),
			zap.String("expected", expectedSignature),
			zap.String("received", signature))
	} else {
		zap.L().Debug("PayOS webhook signature verified successfully",
			zap.String("query_string", queryString))
	}

	return isValid
}

// NewPayOSProxy creates a new PayOS proxy instance
func NewPayOSProxy(httpClient *http.Client, config *config.AppConfig) iproxies.PayOSProxy {
	payosConfig := config.PayOS
	return &payosProxy{
		BaseProxy:   NewBaseProxy(httpClient, payosConfig.BaseURL, config),
		clientID:    payosConfig.ClientID,
		apiKey:      payosConfig.APIKey,
		checksumKey: payosConfig.ChecksumKey,
	}
}
