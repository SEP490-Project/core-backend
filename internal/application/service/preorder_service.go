package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

type preOrderService struct {
	config                       *config.AppConfig
	preOrderRepository           irepository.GenericRepository[model.PreOrder]
	paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
	payOSProxy                   iproxies.PayOSProxy
	shippingAddressRepository    irepository.GenericRepository[model.ShippingAddress]
	ghnService                   iservice_third_party.GHNService
}

func (p preOrderService) PreserverOrder(ctx context.Context, request requests.PreOrderRequest, unitOfWork irepository.UnitOfWork) (*model.PreOrder, error) {
	var preOrder *model.PreOrder
	now := time.Now()

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//1. validate variant and stocks/products
		variant, err := uow.ProductVariant().GetByID(ctx, request.VariantID, []string{"Product"})
		if err != nil {
			return fmt.Errorf("variant %w not found", err)
		} else if err := validateVariantForPreOrder(*variant); err != nil {
			return err
		}

		//1.1 validate shipping address
		address, err := uow.ShippingAddresses().GetByID(ctx, request.AddressID, []string{})
		if err != nil {
			return fmt.Errorf("failed to get shipping address: %w", err)
		}

		// 2. Build persistent model -> minus stock, create pre-order record
		// 2.1 build pre-order model
		preOrder = requests.PreOrderRequest{}.ToModel(*address, *variant, now)
		// 2.2 minus 1 stock
		if variant.CurrentStock == nil {
			return fmt.Errorf("variant stock is nil")
		}
		variant.CurrentStock = ptr.Int(*variant.CurrentStock - 1)
		err = uow.ProductVariant().Update(ctx, variant)
		if err != nil {
			return fmt.Errorf("failed to update variant stock: %w", err)
		}
		// 2.3 create pre-order
		err = uow.PreOrder().Add(ctx, preOrder)
		if err != nil {
			return fmt.Errorf("failed to add preorder: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return preOrder, nil
}

func (p preOrderService) GetPreOrdersByUserIDWithPagination(userID uuid.UUID, limit, page int, search string, status string) ([]model.PreOrder, int, error) {
	ctx := context.Background()

	pageNum := page
	pageSize := limit
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (pageNum - 1) * pageSize

	// Normalize/validate status if provided
	var validStatus *enum.PreOrderStatus
	if status != "" {
		s := enum.PreOrderStatus(status)
		if s.IsValid() {
			validStatus = &s
		}
	}

	// Base filter with user, optional joins for searching by product name/full name, and optional status
	filter := func(db *gorm.DB) *gorm.DB {
		db = db.Where("pre_orders.user_id = ?", userID)
		if validStatus != nil {
			db = db.Where("pre_orders.status = ?", *validStatus)
		}
		if search != "" {
			// join product tables for searching by product name
			db = db.Joins("LEFT JOIN product_variants pv ON pv.id = pre_orders.variant_id").
				Joins("LEFT JOIN products p ON p.id = pv.product_id").
				Where("p.name ILIKE ? OR pre_orders.full_name ILIKE ?", "%"+search+"%", "%"+search+"%")
		}
		return db.Order("pre_orders.created_at DESC").Order("pre_orders.id")
	}

	includes := []string{"ProductVariant", "ProductVariant.Product"}

	// 1) fetch paged IDs first
	var ids []uuid.UUID
	if err := p.preOrderRepository.DB().
		WithContext(ctx).
		Model(&model.PreOrder{}).
		Scopes(filter).
		Select("pre_orders.id").
		Limit(pageSize).
		Offset(offset).
		Pluck("pre_orders.id", &ids).Error; err != nil {
		zap.L().Error("Failed to fetch preorder IDs", zap.Error(err))
		return nil, 0, err
	}

	if len(ids) == 0 {
		return []model.PreOrder{}, 0, nil
	}

	// 2) count total with same criteria but without pagination
	countScope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("pre_orders.user_id = ?", userID)
		if validStatus != nil {
			db = db.Where("pre_orders.status = ?", *validStatus)
		}
		if search != "" {
			db = db.Joins("LEFT JOIN product_variants pv ON pv.id = pre_orders.variant_id").
				Joins("LEFT JOIN products p ON p.id = pv.product_id").
				Where("p.name ILIKE ? OR pre_orders.full_name ILIKE ?", "%"+search+"%", "%"+search+"%")
		}
		return db
	}

	var total int64
	if err := p.preOrderRepository.DB().
		WithContext(ctx).
		Model(&model.PreOrder{}).
		Scopes(countScope).
		Count(&total).Error; err != nil {
		zap.L().Error("Failed to count preorders", zap.Error(err))
		return nil, 0, err
	}

	// 3) final fetch with includes by IDs to avoid duplication
	finalFilter := func(db *gorm.DB) *gorm.DB {
		return db.Where("pre_orders.id IN ?", ids).
			Order("pre_orders.created_at DESC")
	}

	preorders, _, err := p.preOrderRepository.GetAll(ctx, finalFilter, includes, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch preorders with includes", zap.Error(err))
		return nil, 0, err
	}

	return preorders, int(total), nil
}

func (p preOrderService) PayPreOrder(ctx context.Context, orderID uuid.UUID, successURL, cancelURL string, unitOfWork irepository.UnitOfWork) (*model.PaymentTransaction, error) {

	var paymentTransaction *model.PaymentTransaction

	rftCancelURL := fmt.Sprintf("%s?returnUrl=%s", "http://localhost:8080/api/v1/payos/cancel-callback", cancelURL)

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// 1. Preload pre-order
		preOrder, err := uow.PreOrder().GetByID(ctx, orderID, []string{"ProductVariant", "ProductVariant.Product"})
		if err != nil {
			return fmt.Errorf("failed to get pre-order: %w", err)
		}

		// 2. Map pre-order to PayOSItem
		// Create PayOS items (part of payment link)
		payOSItems, total := preOrderToPayOSItemsWithTotalPrice(*preOrder)

		// 3. Prerequisite To Create PayOS Link
		// 3.1 Calculate expiry time
		expirySeconds := p.config.AdminConfig.PayOSLinkExpiry
		if expirySeconds == 0 {
			expirySeconds = 300 // Default 5 minutes
		}
		expiredAt := time.Now().Add(time.Duration(expirySeconds) * time.Second).Unix()

		// 3.2 Setup Signature and Description
		orderCode := GenerateOrderCode()
		description := helper.GeneratePayOSDescription(enum.PaymentTransactionReferenceTypeOrder.String(), orderID)
		signature, err := p.generateSignature(total, rftCancelURL, description, orderCode, successURL)

		if err != nil {
			zap.L().Error("Failed to generate signature", zap.Error(err))
			return fmt.Errorf("failed to generate signature: %w", err)
		}

		// 4. Create PayOS Payment Transaction
		// 4.1 Build CreateLinkRequestDTO to create payment link
		createLinkReq := dtos.PayOSCreateLinkRequest{
			OrderCode:    orderCode,
			Amount:       total,
			Description:  description,
			BuyerName:    utils.PtrOrNil(preOrder.FullName),
			BuyerEmail:   utils.PtrOrNil(preOrder.Email),
			BuyerPhone:   utils.PtrOrNil(preOrder.PhoneNumber),
			BuyerAddress: utils.PtrOrNil(preOrder.AddressLine2),
			Items:        payOSItems,
			CancelURL:    rftCancelURL,
			ReturnURL:    successURL,
			ExpiredAt:    expiredAt, //additional for 5 mins
			Signature:    signature,
		}

		// 4.2 Call PayOS Proxy to create payment transaction
		payosCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		payosResp, err := p.payOSProxy.CreatePaymentLink(payosCtx, &createLinkReq)
		if err != nil {
			// If the error is due to context deadline, return a clear error
			if errors.Is(err, context.DeadlineExceeded) {
				zap.L().Error("PayOS CreatePaymentLink timed out", zap.Error(err), zap.String("order_id", preOrder.ID.String()))
				return fmt.Errorf("payment provider timeout: %w", err)
			}
			zap.L().Error("paymentTransactionService.CreatePaymentLink failed", zap.Error(err), zap.String("order_id", preOrder.ID.String()))
			return err
		}

		// 5. Build PaymentTransaction model
		// Build local PaymentTransaction record and persist
		pt := &model.PaymentTransaction{
			ReferenceID:     preOrder.ID,
			ReferenceType:   "PREORDER",
			Amount:          utils.PtrOrNil(float64(total)),
			Method:          "PAYOS",
			Status:          enum.PaymentTransactionStatusPending,
			TransactionDate: time.Now(),
			PayOSMetadata: &model.PayOSMetadata{
				PaymentLinkID: payosResp.PaymentLinkID,
				OrderCode:     orderCode,
				CheckoutURL:   payosResp.CheckoutURL,
				QRCode:        payosResp.QRCode,
				Bin:           payosResp.Bin,
				AccountNumber: payosResp.AccountNumber,
				AccountName:   payosResp.AccountName,
				ExpiredAt:     payosResp.ExpiredAt,
				Amount:        payosResp.Amount,
				Description:   payosResp.Description,
				Currency:      payosResp.Currency,
			},
			GatewayRef: payosResp.CheckoutURL,
			GatewayID:  payosResp.PaymentLinkID,
		}

		if err := uow.PaymentTransaction().Add(ctx, pt); err != nil {
			zap.L().Error("Failed to save payment transaction", zap.Error(err))
			return err
		}

		paymentTransaction = pt
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paymentTransaction, nil
}

func NewPreOrderService(cfg *config.AppConfig, dbRegistry *gormrepository.DatabaseRegistry, registry *infrastructure.InfrastructureRegistry) iservice.PreOrderService {
	return &preOrderService{
		config:                       cfg,
		preOrderRepository:           dbRegistry.PreOrderRepository,
		paymentTransactionRepository: dbRegistry.PaymentTransactionRepository,
		payOSProxy:                   registry.ProxiesRegistry.PayOSProxy,
		shippingAddressRepository:    dbRegistry.ShippingAddressRepository,
		ghnService:                   registry.GHNService,
	}
}

// ----------------------------Validator-----------------------------//
func validateVariantForPreOrder(variant model.ProductVariant) error {
	//validate if product type is limited
	if variant.Product != nil && variant.Product.Type != enum.ProductTypeLimited {
		return fmt.Errorf("invalid product type for pre-order")
	}

	//validate if product is active
	if variant.Product.IsActive == false {
		return fmt.Errorf("product is not available, please contact admin to activate it")
	}

	//Check stock
	if *variant.CurrentStock == 0 {
		return fmt.Errorf("variant is out of stock")
	}
	return nil
}

//=========== Helper Methods ===========

func (o preOrderService) generateSignature(amount int64, cancelURL, description string, orderCode int64, returnURL string) (string, error) {
	data := fmt.Sprintf(
		"amount=%d&cancelUrl=%s&description=%s&orderCode=%d&returnUrl=%s",
		amount, cancelURL, description, orderCode, returnURL,
	)
	mac := hmac.New(sha256.New, []byte(o.config.PayOS.ChecksumKey))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

//----------------------------Mapper-----------------------------//

// preOrderToPayOSItemsWithTotalPrice maps a PreOrder model to a slice of PayOSItem DTOs.
func preOrderToPayOSItemsWithTotalPrice(preOrder model.PreOrder) ([]dtos.PayOSItem, int64) {
	items := make([]dtos.PayOSItem, 0)
	total := 0

	// Build variant descriptive string (e.g. "250ML - Bottle - Spray")
	variantPropConcat := fmt.Sprintf("%v", preOrder.Capacity) +
		utils.ToTitleCase(*preOrder.CapacityUnit) + " - " +
		utils.ToTitleCase(preOrder.ContainerType.String()) + " - " +
		utils.ToTitleCase(preOrder.DispenserType.String())

	// Build readable item name (e.g. "Shampoo (250ML - Bottle - Spray)")
	variantName := utils.ToTitleCase(preOrder.ProductVariant.Product.Name) + fmt.Sprintf(" (%s)", variantPropConcat)

	mappedModel := dtos.PayOSItem{
		Name:     variantName,
		Quantity: 1,
		Price:    preOrder.UnitPrice,
	}

	items = append(items, mappedModel)
	total += int(preOrder.UnitPrice)

	return items, int64(total)
}
