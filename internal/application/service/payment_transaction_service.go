package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type paymentTransactionService struct {
	paymentTransactionRepo irepository.GenericRepository[model.PaymentTransaction]
	payosProxy             iproxies.PayOSProxy
	config                 *config.AppConfig
}

// GeneratePaymentLink implements iservice.PaymentTransactionService
func (s *paymentTransactionService) GeneratePaymentLink(ctx context.Context, uow irepository.UnitOfWork, req *requests.PaymentRequest) (*responses.PayOSLinkResponse, error) {
	zap.L().Info("Generating PayOS payment link",
		zap.Int64("amount", req.Amount),
		zap.String("reference_id", req.ReferenceID.String()),
		zap.String("reference_type", req.ReferenceType.String()))

	// 1. Create PaymentTransaction record first (to get ID for description)
	paymentTransaction := &model.PaymentTransaction{
		ID:              uuid.New(), // Generate ID before insert
		ReferenceID:     req.ReferenceID,
		ReferenceType:   req.ReferenceType,
		Amount:          utils.PtrOrNil(float64(req.Amount)),
		Method:          "PAYOS",
		Status:          enum.PaymentTransactionStatusPending,
		TransactionDate: time.Now(),
	}

	// 2. Generate order code and description
	orderCode := s.generateOrderCode()
	description := helper.GeneratePayOSDescription(paymentTransaction.ReferenceType.String(), paymentTransaction.ID)

	// 3. Calculate expiry time
	expirySeconds := s.config.AdminConfig.PayOSLinkExpiry
	if expirySeconds == 0 {
		expirySeconds = 300 // Default 5 minutes
	}
	expiredAt := time.Now().Add(time.Duration(expirySeconds) * time.Second).Unix()

	// 4. Generate signature
	signature, err := s.generateSignature(req.Amount, s.config.PayOS.CancelURL, description, orderCode, s.config.PayOS.ReturnURL)
	if err != nil {
		zap.L().Error("Failed to generate signature", zap.Error(err))
		return nil, fmt.Errorf("failed to generate signature: %w", err)
	}

	// 5. Build PayOS request
	payosReq := &dtos.PayOSCreateLinkRequest{
		OrderCode:   orderCode,
		Amount:      req.Amount,
		Description: description,
		BuyerName:   utils.PtrOrNil(req.BuyerName),
		BuyerEmail:  utils.PtrOrNil(req.BuyerEmail),
		BuyerPhone:  utils.PtrOrNil(req.BuyerPhone),
		Items:       s.mapPaymentItems(req.Items),
		CancelURL:   s.config.PayOS.CancelURL,
		ReturnURL:   s.config.PayOS.ReturnURL,
		ExpiredAt:   expiredAt,
		Signature:   signature,
	}

	// 6. Call PayOS API via proxy
	payosResp, err := s.payosProxy.CreatePaymentLink(ctx, payosReq)
	if err != nil {
		zap.L().Error("Failed to create PayOS payment link", zap.Error(err))
		return nil, fmt.Errorf("failed to create payment link: %w", err)
	}

	// 7. Store PayOS metadata
	paymentTransaction.PayOSMetadata = &model.PayOSMetadata{
		PaymentLinkID: payosResp.PaymentLinkID,
		OrderCode:     orderCode,
		CheckoutURL:   payosResp.CheckoutURL,
		QRCode:        payosResp.QRCode,
		Bin:           payosResp.Bin,
		AccountNumber: payosResp.AccountNumber,
		AccountName:   payosResp.AccountName,
		ExpiredAt:     expiredAt,
		Amount:        payosResp.Amount,
		Description:   payosResp.Description,
		Currency:      payosResp.Currency,
	}
	paymentTransaction.GatewayRef = payosResp.CheckoutURL
	paymentTransaction.GatewayID = payosResp.PaymentLinkID

	// 8. Persist to database using UnitOfWork
	if err := uow.PaymentTransaction().Add(ctx, paymentTransaction); err != nil {
		zap.L().Error("Failed to save payment transaction", zap.Error(err))
		// Attempt to cancel the PayOS link since we couldn't save it
		go func() {
			cancelCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// Note: This cancellation happens outside transaction scope
			_ = s.CancelPaymentLink(cancelCtx, nil, strconv.FormatInt(orderCode, 10), "Failed to save transaction")
		}()
		return nil, fmt.Errorf("failed to save payment transaction: %w", err)
	}

	zap.L().Info("PayOS payment link created successfully",
		zap.String("payment_link_id", payosResp.PaymentLinkID),
		zap.Int64("order_code", orderCode),
		zap.String("transaction_id", paymentTransaction.ID.String()))

	return payosResp, nil
}

// GetPaymentStatus implements iservice.PaymentTransactionService
func (s *paymentTransactionService) GetPaymentStatus(ctx context.Context, orderCode string) (*responses.PayOSOrderInfoResponse, error) {
	zap.L().Info("Fetching PayOS payment status", zap.String("order_code", orderCode))

	payosResp, err := s.payosProxy.GetPaymentInfo(ctx, orderCode)
	if err != nil {
		zap.L().Error("Failed to get PayOS payment info", zap.Error(err), zap.String("order_code", orderCode))
		return nil, fmt.Errorf("failed to get payment info: %w", err)
	}

	return payosResp, nil
}

// CancelPaymentLink implements iservice.PaymentTransactionService
func (s *paymentTransactionService) CancelPaymentLink(ctx context.Context, uow irepository.UnitOfWork, orderCode string, reason string) error {
	zap.L().Info("Cancelling PayOS payment link", zap.String("order_code", orderCode), zap.String("reason", reason))

	// 1. Cancel via PayOS API
	payosResp, err := s.payosProxy.CancelPaymentLink(ctx, orderCode, reason)
	if err != nil {
		zap.L().Error("Failed to cancel PayOS payment link", zap.Error(err), zap.String("order_code", orderCode))
		return fmt.Errorf("failed to cancel payment link: %w", err)
	}

	// 2. Find and update local payment transaction (using UnitOfWork if provided, else use repo directly)
	orderCodeInt, _ := strconv.ParseInt(orderCode, 10, 64)
	filterQuery := func(db *gorm.DB) *gorm.DB {
		return db.Where("payos_metadata->>'order_code' = ?", strconv.FormatInt(orderCodeInt, 10))
	}

	var transactions []model.PaymentTransaction

	// Use transactional repo if UnitOfWork provided, otherwise use direct repo
	if uow != nil {
		transactions, _, err = uow.PaymentTransaction().GetAll(ctx, filterQuery, nil, 1, 1)
	} else {
		transactions, _, err = s.paymentTransactionRepo.GetAll(ctx, filterQuery, nil, 1, 1)
	}

	if err != nil {
		zap.L().Error("Failed to find payment transaction", zap.Error(err))
		return fmt.Errorf("failed to find payment transaction: %w", err)
	}

	if len(transactions) == 0 {
		zap.L().Warn("Payment transaction not found for order code", zap.String("order_code", orderCode))
		return nil // PayOS link cancelled but no local record found
	}

	transaction := transactions[0]

	// 3. Update status and metadata
	transaction.Status = enum.PaymentTransactionStatusCancelled
	if transaction.PayOSMetadata != nil {
		now := time.Now()
		transaction.PayOSMetadata.CancelledAt = &now
		transaction.PayOSMetadata.CancellationReason = &reason

		// Update transactions from PayOS response
		if len(payosResp.Transactions) > 0 {
			transaction.PayOSMetadata.Transactions = s.mapPayOSTransactions(payosResp.Transactions)
		}
	}

	// Use transactional repo if UnitOfWork provided
	if uow != nil {
		err = uow.PaymentTransaction().Update(ctx, &transaction)
	} else {
		err = s.paymentTransactionRepo.Update(ctx, &transaction)
	}

	if err != nil {
		zap.L().Error("Failed to update payment transaction status", zap.Error(err))
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	zap.L().Info("Payment link cancelled successfully", zap.String("order_code", orderCode))
	return nil
}

// ProcessWebhook implements iservice.PaymentTransactionService
func (s *paymentTransactionService) ProcessWebhook(ctx context.Context, uow irepository.UnitOfWork, webhookPayload *dtos.PayOSWebhookPayload) error {
	zap.L().Info("Processing PayOS webhook",
		zap.Int64("order_code", webhookPayload.Data.OrderCode),
		zap.String("code", webhookPayload.Code))

	// 1. Find payment transaction by order code using UnitOfWork
	filterQuery := func(db *gorm.DB) *gorm.DB {
		return db.Where("payos_metadata->>'order_code' = ?", strconv.FormatInt(webhookPayload.Data.OrderCode, 10))
	}

	transactions, _, err := uow.PaymentTransaction().GetAll(ctx, filterQuery, nil, 1, 1)
	if err != nil {
		zap.L().Error("Failed to find payment transaction", zap.Error(err))
		return fmt.Errorf("failed to find payment transaction: %w", err)
	}

	if len(transactions) == 0 {
		zap.L().Warn("Payment transaction not found for webhook", zap.Int64("order_code", webhookPayload.Data.OrderCode))
		return fmt.Errorf("payment transaction not found for order code: %d", webhookPayload.Data.OrderCode)
	}

	transaction := transactions[0]

	// 2. Map PayOS status to internal status
	var payosStatus string
	if webhookPayload.Code == "00" {
		payosStatus = "PAID"
	} else {
		payosStatus = webhookPayload.Data.Code
	}

	newStatus := dtos.MapPayOSStatusString(payosStatus)

	// 3. Update transaction status and metadata
	transaction.Status = newStatus

	if transaction.PayOSMetadata != nil {
		// Parse transaction datetime
		transactionTime, _ := time.Parse("2006-01-02 15:04:05", webhookPayload.Data.TransactionDateTime)

		// Add transaction detail to metadata
		payosTransaction := model.PayOSTransaction{
			Amount:                 int(webhookPayload.Data.Amount),
			Description:            webhookPayload.Data.Description,
			AccountNumber:          webhookPayload.Data.AccountNumber,
			Reference:              webhookPayload.Data.Reference,
			TransactionDateTime:    transactionTime,
			CounterAccountBankID:   utils.DerefPtr(webhookPayload.Data.CounterAccountBankID, ""),
			CounterAccountBankName: utils.DerefPtr(webhookPayload.Data.CounterAccountBankName, ""),
			CounterAccountName:     utils.DerefPtr(webhookPayload.Data.CounterAccountName, ""),
			CounterAccountNumber:   utils.DerefPtr(webhookPayload.Data.CounterAccountNumber, ""),
			VirtualAccountName:     utils.DerefPtr(webhookPayload.Data.VirtualAccountName, ""),
			VirtualAccountNumber:   utils.DerefPtr(webhookPayload.Data.VirtualAccountNumber, ""),
		}

		transaction.PayOSMetadata.Transactions = append(transaction.PayOSMetadata.Transactions, payosTransaction)
	}

	// 4. Persist changes using UnitOfWork
	if err := uow.PaymentTransaction().Update(ctx, &transaction); err != nil {
		zap.L().Error("Failed to update payment transaction from webhook", zap.Error(err))
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	zap.L().Info("Webhook processed successfully",
		zap.String("transaction_id", transaction.ID.String()),
		zap.String("new_status", string(newStatus)))

	return nil
}

// ConfirmWebhookURL implements iservice.PaymentTransactionService
func (s *paymentTransactionService) ConfirmWebhookURL(ctx context.Context, webhookURL string) (*dtos.PayOSConfirmWebhookResponse, error) {
	zap.L().Info("Confirming PayOS webhook URL", zap.String("webhook_url", webhookURL))

	response, err := s.payosProxy.ConfirmWebhookURL(ctx, webhookURL)
	if err != nil {
		zap.L().Error("Failed to confirm PayOS webhook URL", zap.Error(err))
		return nil, fmt.Errorf("failed to confirm webhook URL: %w", err)
	}

	zap.L().Info("PayOS webhook URL confirmed successfully", zap.String("webhook_url", webhookURL))
	return response, nil
}

// CancelExpiredLinks implements iservice.PaymentTransactionService
func (s *paymentTransactionService) CancelExpiredLinks(ctx context.Context) (int, error) {
	zap.L().Info("Starting expired payment links cancellation job")

	// Find expired pending PayOS payments
	// Only process records not updated in last 15 minutes to avoid race with webhooks
	cutoffTime := time.Now().Add(-15 * time.Minute)
	now := time.Now().Unix()

	filterQuery := func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", string(enum.PaymentTransactionStatusPending)).
			Where("method = ?", "PAYOS").
			Where("updated_at < ?", cutoffTime).
			Where("payos_metadata->>'expired_at' < ?", strconv.FormatInt(now, 10))
	}

	expiredTransactions, _, err := s.paymentTransactionRepo.GetAll(ctx, filterQuery, nil, 100, 1)
	if err != nil {
		zap.L().Error("Failed to fetch expired payment transactions", zap.Error(err))
		return 0, fmt.Errorf("failed to fetch expired transactions: %w", err)
	}

	if len(expiredTransactions) == 0 {
		zap.L().Info("No expired payment links found")
		return 0, nil
	}

	zap.L().Info("Found expired payment links", zap.Int("count", len(expiredTransactions)))

	cancelledCount := 0
	for _, transaction := range expiredTransactions {
		if transaction.PayOSMetadata == nil {
			continue
		}

		orderCode := strconv.FormatInt(transaction.PayOSMetadata.OrderCode, 10)

		// Try to cancel via PayOS API (without UnitOfWork since this is a batch job)
		if err := s.CancelPaymentLink(ctx, nil, orderCode, "Expired payment link"); err != nil {
			zap.L().Warn("Failed to cancel expired payment link",
				zap.String("order_code", orderCode),
				zap.Error(err))
			continue
		}

		cancelledCount++
	}

	zap.L().Info("Expired payment links cancellation completed",
		zap.Int("total_found", len(expiredTransactions)),
		zap.Int("cancelled", cancelledCount))

	return cancelledCount, nil
}

// SyncPaymentStatus implements iservice.PaymentTransactionService
func (s *paymentTransactionService) SyncPaymentStatus(ctx context.Context, uow irepository.UnitOfWork, paymentTransactionID uuid.UUID) error {
	zap.L().Info("Syncing payment status", zap.String("transaction_id", paymentTransactionID.String()))

	// 1. Fetch local transaction using UnitOfWork
	transaction, err := uow.PaymentTransaction().GetByID(ctx, paymentTransactionID, nil)
	if err != nil {
		zap.L().Error("Failed to fetch payment transaction", zap.Error(err))
		return fmt.Errorf("failed to fetch transaction: %w", err)
	}

	if transaction == nil {
		return fmt.Errorf("payment transaction not found: %s", paymentTransactionID)
	}

	if transaction.PayOSMetadata == nil {
		return fmt.Errorf("payment transaction has no PayOS metadata")
	}

	// 2. Fetch latest status from PayOS
	orderCode := strconv.FormatInt(transaction.PayOSMetadata.OrderCode, 10)
	payosResp, err := s.payosProxy.GetPaymentInfo(ctx, orderCode)
	if err != nil {
		zap.L().Error("Failed to get PayOS payment info", zap.Error(err))
		return fmt.Errorf("failed to get payment info: %w", err)
	}

	// 3. Map and update status
	newStatus := dtos.MapPayOSStatusString(payosResp.Status)
	transaction.Status = newStatus

	// 4. Update metadata with latest info
	if transaction.PayOSMetadata != nil {
		transaction.PayOSMetadata.Transactions = s.mapPayOSTransactions(payosResp.Transactions)

		if payosResp.CanceledAt.Unix() > 0 {
			transaction.PayOSMetadata.CancelledAt = &payosResp.CanceledAt
			transaction.PayOSMetadata.CancellationReason = &payosResp.CancellationReason
		}
	}

	// 5. Persist changes using UnitOfWork
	if err := uow.PaymentTransaction().Update(ctx, transaction); err != nil {
		zap.L().Error("Failed to update payment transaction", zap.Error(err))
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	zap.L().Info("Payment status synced successfully",
		zap.String("transaction_id", paymentTransactionID.String()),
		zap.String("new_status", string(newStatus)))

	return nil
}

// region: =========== Helper Methods ===========

func (s *paymentTransactionService) generateOrderCode() int64 {
	now := time.Now().Unix()
	randPart := time.Now().UnixNano() % 1e3
	return now*1000 + randPart
}

func (s *paymentTransactionService) generateSignature(amount int64, cancelURL, description string, orderCode int64, returnURL string) (string, error) {
	data := fmt.Sprintf(
		"amount=%d&cancelUrl=%s&description=%s&orderCode=%d&returnUrl=%s",
		amount, cancelURL, description, orderCode, returnURL,
	)
	mac := hmac.New(sha256.New, []byte(s.config.PayOS.ChecksumKey))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (s *paymentTransactionService) mapPaymentItems(items []requests.PaymentItemRequest) []dtos.PayOSItem {
	payosItems := make([]dtos.PayOSItem, 0, len(items))
	for _, item := range items {
		payosItems = append(payosItems, dtos.PayOSItem{
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    float64(item.Price),
		})
	}
	return payosItems
}

func (s *paymentTransactionService) mapPayOSTransactions(transactions []struct {
	Amount                 int       `json:"amount"`
	Description            string    `json:"description"`
	AccountNumber          string    `json:"accountNumber"`
	Reference              string    `json:"reference"`
	TransactionDateTime    time.Time `json:"transactionDateTime"`
	CounterAccountBankID   string    `json:"counterAccountBankId"`
	CounterAccountBankName string    `json:"counterAccountBankName"`
	CounterAccountName     string    `json:"counterAccountName"`
	CounterAccountNumber   string    `json:"counterAccountNumber"`
	VirtualAccountName     string    `json:"virtualAccountName"`
	VirtualAccountNumber   string    `json:"virtualAccountNumber"`
}) []model.PayOSTransaction {
	result := make([]model.PayOSTransaction, 0, len(transactions))
	for _, tx := range transactions {
		result = append(result, model.PayOSTransaction{
			Amount:                 tx.Amount,
			Description:            tx.Description,
			AccountNumber:          tx.AccountNumber,
			Reference:              tx.Reference,
			TransactionDateTime:    tx.TransactionDateTime,
			CounterAccountBankID:   tx.CounterAccountBankID,
			CounterAccountBankName: tx.CounterAccountBankName,
			CounterAccountName:     tx.CounterAccountName,
			CounterAccountNumber:   tx.CounterAccountNumber,
			VirtualAccountName:     tx.VirtualAccountName,
			VirtualAccountNumber:   tx.VirtualAccountNumber,
		})
	}
	return result
}

// endregion

// NewPaymentTransactionService creates a new PaymentTransactionService instance
func NewPaymentTransactionService(
	paymentTransactionRepo irepository.GenericRepository[model.PaymentTransaction],
	payosProxy iproxies.PayOSProxy,
) iservice.PaymentTransactionService {
	return &paymentTransactionService{
		paymentTransactionRepo: paymentTransactionRepo,
		payosProxy:             payosProxy,
		config:                 config.GetAppConfig(),
	}
}
