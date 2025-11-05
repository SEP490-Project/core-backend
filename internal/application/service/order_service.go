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
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type orderService struct {
	config                       *config.AppConfig
	orderRepository              irepository.GenericRepository[model.Order]
	orderItemRepository          irepository.GenericRepository[model.OrderItem]
	paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
	payOSProxy                   iproxies.PayOSProxy
	shippingAddressRepository    irepository.GenericRepository[model.ShippingAddress]
	ghnService                   iservice_third_party.GHNService
}

func (o *orderService) GetOrdersByUserIDWithPagination(userID uuid.UUID, limit, page int, search string) ([]model.Order, int, error) {
	ctx := context.Background()

	pageNum := page
	pageSize := limit
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (pageNum - 1) * pageSize

	filter := func(db *gorm.DB) *gorm.DB {
		db = db.Where("orders.user_id = ?", userID)
		if search != "" {
			db = db.Where("orders.order_no ILIKE ?", "%"+search+"%")
		}
		return db.Order("orders.created_at DESC").Order("orders.id")
	}

	includes := []string{"OrderItems"}

	var orderIDs []uuid.UUID
	err := o.orderRepository.DB().
		WithContext(ctx).
		Model(&model.Order{}).
		Scopes(filter).
		Select("orders.id").
		Limit(pageSize).
		Offset(offset).
		Pluck("orders.id", &orderIDs).Error
	if err != nil {
		zap.L().Error("Failed to fetch order IDs", zap.Error(err))
		return nil, 0, err
	}

	if len(orderIDs) == 0 {
		return []model.Order{}, 0, nil
	}

	countScope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("orders.user_id = ?", userID)
		if search != "" {
			db = db.Where("orders.order_no ILIKE ?", "%"+search+"%")
		}
		return db
	}
	var total int64
	if err := o.orderRepository.DB().
		WithContext(ctx).
		Model(&model.Order{}).
		Scopes(countScope).
		Count(&total).Error; err != nil {
		zap.L().Error("Failed to count orders", zap.Error(err))
		return nil, 0, err
	}

	finalFilter := func(db *gorm.DB) *gorm.DB {
		return db.Where("orders.id IN ?", orderIDs).
			Order("orders.created_at DESC")
	}

	orders, _, err := o.orderRepository.GetAll(ctx, finalFilter, includes, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch orders with includes", zap.Error(err))
		return nil, 0, err
	}

	return orders, int(total), nil
}

func (o *orderService) PlaceOrder(ctx context.Context, userID uuid.UUID, request requests.OrderRequest, shippingPrice int, unitOfWork irepository.UnitOfWork) (*model.Order, error) {
	now := time.Now()
	var persistedOrder *model.Order

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//Create Item
		var persistedOrderItem []model.OrderItem

		for _, item := range request.Items {
			//check variantID:
			includes := []string{"AttributeValues", "AttributeValues.Attribute", "Images"}
			variant, err := uow.ProductVariant().GetByID(ctx, item.VariantID, includes)
			if err != nil {
				zap.L().Error("ProductVariant().GetByID", zap.Error(err))
				return errors.New("Product Variant not found")
			} else if variant == nil {
				return errors.New("Product Variant not found")
			}
			persistedItem := item.ToModel(*variant, now)
			persistedOrderItem = append(persistedOrderItem, *persistedItem)
		}

		//Create Order
		shippingAddress, err := o.shippingAddressRepository.GetByID(ctx, request.AddressID, nil)
		if err != nil {
			zap.L().Error("ShippingAddress().GetByID", zap.Error(err))
			return err
		}

		persistedOrder = request.ToModel(userID, persistedOrderItem, *shippingAddress, shippingPrice, now)
		err = uow.Order().Add(ctx, persistedOrder)
		if err != nil {
			zap.L().Error("Order().Add", zap.Error(err))
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return persistedOrder, nil
}

// PayOrder handles the payment process in a atomic transaction
func (o *orderService) PayOrder(ctx context.Context, orderID uuid.UUID, shippingFee int, successURL, cancelURL string, unitOfWork irepository.UnitOfWork) (*model.PaymentTransaction, error) {
	var paymentTransaction *model.PaymentTransaction

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//Check Order
		includes := []string{"User", "OrderItems", "OrderItems.Variant", "OrderItems.Variant.Product"}
		order, err := uow.Order().GetByID(ctx, orderID, includes)
		if err != nil {
			return err
		}

		payOSItems, total, err := toPayOSItemsWithTotalPrice(order.OrderItems)
		if err != nil {
			zap.L().Error("Failed to map order items to PayOS items", zap.Error(err))
			return err
		}

		//Add shipping fee item as additional info
		shippingFeeItem := dtos.PayOSItem{
			Name:     "Shipping Fee from \"Giao Hàng Nhanh\"",
			Quantity: 1,
			Price:    float64(shippingFee),
		}
		//Calc total ammount
		amount := total + shippingFee

		payOSItems = append(payOSItems, shippingFeeItem)

		//Calculate expiry time
		expirySeconds := o.config.AdminConfig.PayOSLinkExpiry
		if expirySeconds == 0 {
			expirySeconds = 300 // Default 5 minutes
		}
		expiredAt := time.Now().Add(time.Duration(expirySeconds) * time.Second).Unix()

		//generate signature
		orderCode := o.generateOrderCode()
		description := helper.GeneratePayOSDescription(enum.PaymentTransactionReferenceTypeOrder.String(), orderID)
		signature, err := o.generateSignature(int64(amount), cancelURL, description, orderCode, successURL)

		if err != nil {
			zap.L().Error("Failed to generate signature", zap.Error(err))
			return fmt.Errorf("failed to generate signature: %w", err)
		}

		createLinkReq := dtos.PayOSCreateLinkRequest{
			OrderCode:    orderCode,
			Amount:       int64(total + shippingFee),
			Description:  description,
			BuyerName:    utils.PtrOrNil(order.FullName),
			BuyerEmail:   utils.PtrOrNil(order.Email),
			BuyerPhone:   utils.PtrOrNil(order.PhoneNumber),
			BuyerAddress: utils.PtrOrNil(order.AddressLine2),
			Items:        payOSItems,
			CancelURL:    cancelURL,
			ReturnURL:    successURL,
			ExpiredAt:    expiredAt, //additional for 5 mins
			Signature:    signature,
		}

		// Call PayOS and persist local payment transaction using UnitOfWork
		// Use a short timeout for external call so we don't block forever and to avoid being canceled by Gin timeout
		payosCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		payosResp, err := o.payOSProxy.CreatePaymentLink(payosCtx, &createLinkReq)
		if err != nil {
			// If the error is due to context deadline, return a clear error
			if errors.Is(err, context.DeadlineExceeded) {
				zap.L().Error("PayOS CreatePaymentLink timed out", zap.Error(err), zap.String("order_id", order.ID.String()))
				return fmt.Errorf("payment provider timeout: %w", err)
			}
			zap.L().Error("paymentTransactionService.CreatePaymentLink failed", zap.Error(err), zap.String("order_id", order.ID.String()))
			return err
		}

		// Build local PaymentTransaction record and persist
		pt := &model.PaymentTransaction{
			ID:              uuid.New(),
			ReferenceID:     order.ID,
			ReferenceType:   "ORDER",
			Amount:          utils.PtrOrNil(float64(amount)),
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

		// set the paymentTransaction to the created record for returning later
		paymentTransaction = pt

		return nil
	})

	if err != nil {
		return nil, err
	}

	return paymentTransaction, nil
}

func NewOrderService(cfg *config.AppConfig, dbRegistry *gormrepository.DatabaseRegistry, registry *infrastructure.InfrastructureRegistry) iservice.OrderService {
	return &orderService{
		config:                    cfg,
		orderRepository:           dbRegistry.OrderRepository,
		orderItemRepository:       dbRegistry.OrderItemRepository,
		shippingAddressRepository: dbRegistry.ShippingAddressRepository,
		payOSProxy:                registry.ProxiesRegistry.PayOSProxy,
		ghnService:                registry.GHNService,
	}
}

// region: =========== Helper Methods ===========

func (o *orderService) generateOrderCode() int64 {
	now := time.Now().Unix()
	randPart := time.Now().UnixNano() % 1e3
	return now*1000 + randPart
}

func (o *orderService) generateSignature(amount int64, cancelURL, description string, orderCode int64, returnURL string) (string, error) {
	data := fmt.Sprintf(
		"amount=%d&cancelUrl=%s&description=%s&orderCode=%d&returnUrl=%s",
		amount, cancelURL, description, orderCode, returnURL,
	)
	mac := hmac.New(sha256.New, []byte(o.config.PayOS.ChecksumKey))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// ToPayOSItemsWithTotalPrice map OrderItems to PayOSItems with total price
func toPayOSItemsWithTotalPrice(items []model.OrderItem) (payOSItems []dtos.PayOSItem, total int, err error) {
	payOSItems = make([]dtos.PayOSItem, 0, len(items))
	total = 0

	for _, item := range items {
		// Validate relations: Variant and Product must be preloaded
		if item.Variant.ID == uuid.Nil || item.Variant.Product.ID == uuid.Nil {
			err = fmt.Errorf("missing relation: OrderItem.Variant or OrderItem.Variant.Product not loaded for OrderItem ID %s", item.ID)
			return
		}

		// Build variant descriptive string (e.g. "250ML - Bottle - Spray")
		variantPropConcat := fmt.Sprintf("%v", item.Capacity) +
			utils.ToTitleCase(*item.CapacityUnit) + " - " +
			utils.ToTitleCase(item.ContainerType.String()) + " - " +
			utils.ToTitleCase(item.DispenserType.String())

		// Build readable item name (e.g. "Shampoo (250ML - Bottle - Spray)")
		variantName := utils.ToTitleCase(item.Variant.Product.Name) + fmt.Sprintf(" (%s)", variantPropConcat)

		// Append to PayOS item list
		payOSItems = append(payOSItems, dtos.PayOSItem{
			Name:     variantName,
			Quantity: item.Quantity,
			Price:    item.UnitPrice,
		})

		// Accumulate total price (unit price × quantity)
		total += int(item.UnitPrice) * item.Quantity
	}

	return
}
