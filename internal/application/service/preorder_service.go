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
	notiBuilder "core-backend/internal/application/service/notification_builder"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/state/preordersm"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type preOrderService struct {
	config *config.AppConfig
	db     *gorm.DB
	// preOrderRepository           irepository.GenericRepository[model.PreOrder]
	preOrderRepository           irepository.PreOrderRepository
	paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
	userRepository               irepository.GenericRepository[model.User]
	variantRepository            irepository.GenericRepository[model.ProductVariant]
	payOSProxy                   iproxies.PayOSProxy
	shippingAddressRepository    irepository.GenericRepository[model.ShippingAddress]
	ghnService                   iproxies.GHNProxy
	paymentTransactionService    iservice.PaymentTransactionService
	stateTransferService         iservice.StateTransferService
	notificationService          iservice.NotificationService
}

func (p preOrderService) ApproveRefundRequest(ctx context.Context, preOrderID, actionBy uuid.UUID, reason, fileURL *string) error {
	preOrder, limitedProduct, user, err := p.lookupPreOrderWithLimitedProductAndUser(ctx, preOrderID, actionBy)
	if err != nil {
		return err
	}
	err = MovePreOrderStateUsingFSM(preOrder, limitedProduct, user, enum.PreOrderStatusRefunded, reason)
	if err != nil {
		return err
	}

	preOrder.StaffResource = fileURL
	if err := p.preOrderRepository.Update(ctx, preOrder); err != nil {
		return err
	}

	//Notification
	go func() {
		err = p.sendNotification(context.Background(), notiBuilder.PreOrderNotifyRefund, preOrder, user)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()
	return nil
}

func (p preOrderService) RefundRequest(ctx context.Context, preOrderID, actionBy uuid.UUID, reason *string) error {
	preOrder, limitedProduct, user, err := p.lookupPreOrderWithLimitedProductAndUser(ctx, preOrderID, actionBy)
	if err != nil {
		return err
	}
	err = MovePreOrderStateUsingFSM(preOrder, limitedProduct, user, enum.PreOrderStatusRefundRequest, reason)
	if err != nil {
		return err
	}
	if err := p.preOrderRepository.Update(ctx, preOrder); err != nil {
		return err
	}
	//Notification
	go func() {
		err = p.sendNotification(context.Background(), notiBuilder.PreOrderNotifyRefundRequest, preOrder, user)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()
	return nil
}

func (p preOrderService) ObligateRefund(ctx context.Context, preOrderID, actionBy uuid.UUID, reason, fileURL *string) error {
	preOrder, limitedProduct, user, err := p.lookupPreOrderWithLimitedProductAndUser(ctx, preOrderID, actionBy)
	if err != nil {
		return err
	}
	err = MovePreOrderStateUsingFSM(preOrder, limitedProduct, user, enum.PreOrderStatusRefunded, reason)
	if err != nil {
		return err
	}
	preOrder.StaffResource = fileURL
	if err := p.preOrderRepository.Update(ctx, preOrder); err != nil {
		return err
	}

	//Notification
	go func() {
		err = p.sendNotification(context.Background(), notiBuilder.PreOrderNotifyObligateRefund, preOrder, user)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()

	return nil
}

func (p preOrderService) MarkPreOrderAsReceived(ctx context.Context, preOrderID, updatedBy uuid.UUID) error {
	err := p.stateTransferService.MovePreOrderToState(ctx, preOrderID, enum.PreOrderStatusReceived, updatedBy, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (p preOrderService) RequestCompensation(ctx context.Context, preOrderID, actionBy uuid.UUID, reason, fileURL *string) error {
	preOrder, limitedProduct, user, err := p.lookupPreOrderWithLimitedProductAndUser(ctx, preOrderID, actionBy)
	if err != nil {
		return err
	}

	//Check if compensation has already been requested
	if preOrder.ActionNotes != nil {
		for _, note := range *preOrder.ActionNotes {
			if note.ActionType == enum.PreOrderStatusCompensateRequest {
				return errors.New("you've already requested compensation for this preorder")
			}
		}
	}

	err = MovePreOrderStateUsingFSM(preOrder, limitedProduct, user, enum.PreOrderStatusCompensateRequest, reason)
	if err != nil {
		return err
	}
	preOrder.UserResource = fileURL
	if err := p.preOrderRepository.Update(ctx, preOrder); err != nil {
		zap.L().Error("Failed to update PreOrder state", zap.String("preorder_id", preOrderID.String()), zap.Error(err))
		return errors.New("failed to update PreOrder state: " + err.Error())
	}

	// Notification
	go func() {
		err = p.sendNotification(context.Background(), notiBuilder.PreOrderNotifyCompensateRequested, preOrder, user)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()

	return nil
}

func (p preOrderService) ProcessCompensation(ctx context.Context, preOrderID, actionBy uuid.UUID, isApproved bool, reason, fileURL *string) error {
	preOrder, limitedProduct, user, err := p.lookupPreOrderWithLimitedProductAndUser(ctx, preOrderID, actionBy)
	if err != nil {
		return err
	}
	var notiStatus notiBuilder.PreOrderNotificationType

	if isApproved {
		err = MovePreOrderStateUsingFSM(preOrder, limitedProduct, user, enum.PreOrderStatusCompensated, reason)
		if err != nil {
			return err
		}
		if fileURL == nil {
			return errors.New("file can not be empty if the compensate request being approved")
		}
		preOrder.StaffResource = fileURL
		notiStatus = notiBuilder.PreOrderNotifyCompensated
	} else {
		err = MovePreOrderStateUsingFSM(preOrder, limitedProduct, user, enum.PreOrderStatusDelivered, reason)
		if err != nil {
			return err
		}
		if fileURL != nil {
			preOrder.StaffResource = fileURL
		}
		notiStatus = notiBuilder.PreOrderNotifyCompensateDenied
	}

	if err := p.preOrderRepository.Update(ctx, preOrder); err != nil {
		zap.L().Error("Failed to update PreOrder state", zap.String("preorder_id", preOrderID.String()), zap.Error(err))
		return errors.New("failed to update PreOrder state: " + err.Error())
	}

	// Notification
	go func() {
		err = p.sendNotification(context.Background(), notiStatus, preOrder, user)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()

	return nil
}

func (p preOrderService) PreOrderOpeningChecker(ctx context.Context) (int, int, int) {
	var totalProcessed, failed, upCommingItems = 0, 0, 0
	// 0. Count all PAID items
	countFilter := func(db *gorm.DB) *gorm.DB {
		return db.Joins("JOIN product_variants pv ON pv.id = pre_orders.variant_id").
			Joins("JOIN products p ON p.id = pv.product_id").
			Joins("JOIN limited_products lp ON lp.id = p.id").
			Where("pre_orders.status = ?", enum.PreOrderStatusPreOrdered).
			Where("lp.availability_start_date > ?", time.Now())
	}
	countItem, err := p.preOrderRepository.Count(ctx, countFilter)
	if err != nil {
		zap.L().Error("Failed to count upcomming pre-orders", zap.Error(err))
		return 0, 0, 0
	}
	upCommingItems = int(countItem)

	// 1. Get all pre-orders PRE_ORDERED where current time is after AvailabilityStartDate
	now := time.Now()
	filter := func(db *gorm.DB) *gorm.DB {
		return db.Joins("JOIN product_variants pv ON pv.id = pre_orders.variant_id").
			Joins("JOIN products p ON p.id = pv.product_id").
			Joins("JOIN limited_products lp ON lp.id = p.id").
			Where("pre_orders.status = ?", enum.PreOrderStatusPreOrdered).
			Where("lp.availability_start_date <= ?", now)
	}

	preOrders, total, err := p.preOrderRepository.GetAll(ctx, filter, []string{}, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch pre-orders for opening checker", zap.Error(err))
		return 0, 0, 0
	}
	totalProcessed = int(total)
	_ = preOrders
	for _, preOrder := range preOrders {
		var err error
		if preOrder.IsSelfPickedUp {
			err = p.stateTransferService.MovePreOrderToState(ctx, preOrder.ID, enum.PreOrderStatusAwaitingPickup, uuid.UUID{}, nil, nil)
		} else {
			err = p.stateTransferService.MovePreOrderToState(ctx, preOrder.ID, enum.PreOrderStatusInTransit, uuid.UUID{}, nil, nil)
		}
		if err != nil {
			msg := fmt.Sprintf("Failed to update pre-order status with ID: %s , Detail: %s", preOrder.ID.String(), err.Error())
			zap.L().Error(msg)
			failed++
		}
	}

	return totalProcessed, failed, upCommingItems
}

// validateAndGetRemainingOrder returns remaining allowed pre-order slots for a user+variant.
// It sums quantities in the DB (excluding CANCELLED) to avoid loading all records.
func (p preOrderService) validateAndGetRemainingOrder(ctx context.Context, userID, variantID string, maximumOrder int) (int, error) {
	if maximumOrder <= 0 {
		return 0, fmt.Errorf("invalid maximumOrder: %d", maximumOrder)
	}

	exclude := []enum.PreOrderStatus{
		enum.PreOrderStatusCancelled,
		enum.PreOrderStatusRefunded,
		enum.PreOrderStatusCompensated,
	}

	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ? AND variant_id = ? AND status NOT IN (?)", userID, variantID, exclude)
	}
	preorders, _, err := p.preOrderRepository.GetAll(ctx, filter, []string{}, 0, 0)
	if err != nil {
		return 0, err
	}

	totalHadBought := 0
	for _, preorder := range preorders {
		totalHadBought += preorder.Quantity
	}

	remaining := maximumOrder - int(totalHadBought)
	if remaining <= 0 {
		return 0, fmt.Errorf("you have reached the maximum number of pre-orders allowed for this product variant")
	}
	return remaining, nil
}

func (p preOrderService) PreserverOrder(ctx context.Context, request requests.PreOrderRequest, unitOfWork irepository.UnitOfWork, userID uuid.UUID) (*model.PreOrder, error) {
	var preOrder *model.PreOrder
	var remainedPreservableItem int
	now := time.Now()

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// Validate user bank account
		creator, err := uow.Users().GetByID(ctx, userID, nil)
		if err != nil {
			return err
		}
		if creator.BankAccount == nil || creator.BankName == nil || creator.BankAccountHolder == nil {
			return fmt.Errorf("user bank information is incomplete, please update your bank details before placing a pre-order")
		}

		//1. validate variant and stocks/products
		includes := []string{"Product", "Product.Limited", "Product.Brand", "Product.Category"}
		variant, err := uow.ProductVariant().GetByID(ctx, request.VariantID, includes)
		if err != nil {
			return fmt.Errorf("variant %w not found", err)
		} else if err = ValidateVariantForPreOrder(*variant); err != nil {
			return err
		}

		// Check if this preorderable?
		if variant.Product.Limited != nil {
			// check if user already bought it
			limitItemQuantity := variant.Product.Limited.AchievableQuantity
			remainedPreservableItem, err = p.validateAndGetRemainingOrder(ctx, userID.String(), variant.ID.String(), limitItemQuantity)
			if err != nil {
				return err
			}
			//Check if can user buy this remained Item?
			if request.Quantity > remainedPreservableItem {
				return fmt.Errorf("you can only preorder %d more of this product variant", remainedPreservableItem)
			}

			// check date's validity
			premiereDate := variant.Product.Limited.PremiereDate
			startDate := variant.Product.Limited.AvailabilityStartDate
			endDate := variant.Product.Limited.AvailabilityEndDate
			//isPreOrderable := now.After(premiereDate) && now.Before(startDate) && now.Before(endDate)
			// PREORDER RANGE: [premiereDate, startDate)
			isPreOrderable := !now.Before(premiereDate) && now.Before(startDate)
			if !isPreOrderable {
				// not ready
				if now.Before(premiereDate) {
					remaining := premiereDate.Sub(now)
					return fmt.Errorf("the product is not ready to preorder, time remaining: %s", remaining)
				}

				// already overdue
				if now.After(endDate) || now.Equal(endDate) {
					return fmt.Errorf(
						"the product is overdue to preorder, last available date was %s",
						endDate.Format("2006-01-02 15:04:05"))
				}

				return fmt.Errorf("product is not available for pre-order at this time")
			}

			if variant.PreOrderLimit == variant.PreOrderCount {
				return fmt.Errorf("pre-order limit reached for this variant")
			}

			//Check current orderable stats
			if variant.PreOrderLimit == nil || variant.PreOrderCount == nil {
				zap.L().Debug("Invalid data format: PreOrderLimit or PreOrderCount is nil")
				return fmt.Errorf("pre-order limit or PreOrderCount is nil")
			}

			remainingOrderSlot := *variant.PreOrderLimit - *variant.PreOrderCount
			if remainingOrderSlot <= 0 {
				return fmt.Errorf("no remaining pre-order slots for this variant")
			}
		}

		//1.1 validate shipping address
		address, err := uow.ShippingAddresses().GetByID(ctx, request.AddressID, []string{})
		if err != nil {
			return fmt.Errorf("failed to get shipping address: %w", err)
		}

		// 2. Build persistent model -> minus stock, create pre-order record
		// 2.1 build pre-order model
		preOrder = requests.PreOrderRequest{}.ToModel(*creator, *address, *variant, now, remainedPreservableItem)
		// 2.2 minus 1 stock
		if variant.CurrentStock == nil {
			return fmt.Errorf("variant stock is nil")
		}
		variant.CurrentStock = ptr.Int(*variant.CurrentStock - 1)
		variant.PreOrderCount = ptr.Int(*variant.PreOrderCount + 1)
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

func (s *preOrderService) GetPreOrdersByUserIDWithPagination(
	ctx context.Context,
	userID uuid.UUID,
	limit, page int,
	search string,
	statuses []string,
	createdFrom, createdTo string,
) ([]responses.PreOrderResponse, int, error) {

	// -------------------------------
	// Pagination
	// -------------------------------
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	includes := []string{
		"ProductVariant.Images",
		"Brand",
		"Category",
	}

	// ---- FILTER ----
	filter := func(db *gorm.DB) *gorm.DB {
		q := db.Where("user_id = ?", userID)

		if len(statuses) > 0 {
			q = q.Where("status IN ?", statuses)
		}

		if search != "" {
			like := "%" + search + "%"
			q = q.Where("(product_name ILIKE ? OR email ILIKE ? OR full_name ILIKE ?)", like, like, like)
		}

		if createdFrom != "" {
			q = q.Where("created_at >= ?", createdFrom)
		}

		if createdTo != "" {
			q = q.Where("created_at <= ?", createdTo)
		}

		return q.Order("created_at DESC")
	}

	// ---- LẤY DATA ----
	preOrders, total, err := s.preOrderRepository.GetAll(ctx, filter, includes, limit, page)
	if err != nil {
		return nil, 0, err
	}

	// ---- TỐI ƯU: lấy tất cả PaymentTransaction newest bằng 1 query ----
	ids := make([]uuid.UUID, 0, len(preOrders))
	for _, po := range preOrders {
		ids = append(ids, po.ID)
	}

	var payments []model.PaymentTransaction

	if len(ids) > 0 {
		s.db.
			Raw(`
                SELECT DISTINCT ON (reference_id) *
                FROM payment_transactions
                WHERE reference_id IN ?
                ORDER BY reference_id, created_at DESC
            `, ids).
			Scan(&payments)
	}

	// ---- MAP payments vào preorder ----
	pmMap := map[uuid.UUID]*model.PaymentTransaction{}
	for _, pm := range payments {
		pmCopy := pm
		pmMap[pm.ReferenceID] = &pmCopy
	}

	resp := make([]responses.PreOrderResponse, 0, len(preOrders))
	for _, po := range preOrders {
		resp = append(resp, responses.PreOrderResponse{}.ToPreOrderResponse(po, pmMap[po.ID]))
	}

	return resp, int(total), nil
}

func (p preOrderService) PayForPreservationSlot(ctx context.Context, preOrderID uuid.UUID, returnURL, cancelURL string, unitOfWork irepository.UnitOfWork) (*responses.PayOSLinkResponse, error) {
	var paymentTransaction *responses.PayOSLinkResponse

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//*1. Preload pre-order
		preOrder, err := uow.PreOrder().GetByID(ctx, preOrderID, []string{"ProductVariant", "ProductVariant.Product"})
		if err != nil {
			return fmt.Errorf("failed to get pre-order: %w", err)
		}

		//*2 Build Payment Request:
		paymentItemRequest, total := toPaymentItemRequestWithTotalPrice(*preOrder)
		paymentRq := requests.PaymentRequest{
			ReferenceID:   preOrderID,
			ReferenceType: enum.PaymentTransactionReferenceTypePreOrder,
			PayerID:       &preOrder.UserID,
			Amount:        total,
			Description:   fmt.Sprintf("Payment for preservation %s", preOrder.ID),
			Items:         paymentItemRequest,
			BuyerName:     preOrder.FullName,
			BuyerEmail:    preOrder.Email,
			BuyerPhone:    preOrder.PhoneNumber,
			ReturnURL:     &returnURL,
			CancelURL:     &cancelURL,
		}

		//*3. Create Payment Transaction
		paymentTransaction, err = p.paymentTransactionService.GeneratePaymentLink(ctx, uow, &paymentRq)
		if err != nil {
			zap.L().Error("Failed to initiate payment transaction for order", zap.Error(err))
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paymentTransaction, nil
}

func NewPreOrderService(cfg *config.AppConfig, dbRegistry *gormrepository.DatabaseRegistry, registry *infrastructure.InfrastructureRegistry, paymentTransactionSvc iservice.PaymentTransactionService, stateTransferService iservice.StateTransferService, notificationService iservice.NotificationService) iservice.PreOrderService {
	return &preOrderService{
		config:                       cfg,
		db:                           dbRegistry.GormDatabase,
		preOrderRepository:           dbRegistry.PreOrderRepository,
		paymentTransactionRepository: dbRegistry.PaymentTransactionRepository,
		userRepository:               dbRegistry.UserRepository,
		variantRepository:            dbRegistry.ProductVariantRepository,
		payOSProxy:                   registry.ProxiesRegistry.PayOSProxy,
		shippingAddressRepository:    dbRegistry.ShippingAddressRepository,
		ghnService:                   registry.ProxiesRegistry.GHNProxy,
		paymentTransactionService:    paymentTransactionSvc,
		stateTransferService:         stateTransferService,
		notificationService:          notificationService,
	}
}

// ----------------------------Validator-----------------------------//
func ValidateVariantForPreOrder(variant model.ProductVariant) error {
	//validate if product type is limited
	if variant.Product != nil && variant.Product.Type != enum.ProductTypeLimited {
		return fmt.Errorf("invalid product type for pre-order")
	}

	//validate if product is active
	if !variant.Product.IsActive {
		return fmt.Errorf("product is not available, please contact admin to activate it")
	}

	//Check stock
	if *variant.CurrentStock == 0 {
		return fmt.Errorf("variant is out of stock")
	}
	return nil
}

//=========== Helper Methods ===========

func (p preOrderService) generateSignature(amount int64, cancelURL, description string, orderCode int64, returnURL string) (string, error) {
	data := fmt.Sprintf(
		"amount=%d&cancelUrl=%s&description=%s&orderCode=%d&returnUrl=%s",
		amount, cancelURL, description, orderCode, returnURL,
	)
	mac := hmac.New(sha256.New, []byte(p.config.PayOS.ChecksumKey))
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

func toPaymentItemRequestWithTotalPrice(preOrder model.PreOrder) ([]requests.PaymentItemRequest, int64) {
	items := make([]requests.PaymentItemRequest, 0)
	total := int64(0)

	// Build variant descriptive string (e.g. "250ML - Bottle - Spray")
	variantPropConcat := fmt.Sprintf("%v", preOrder.Capacity) +
		utils.ToTitleCase(*preOrder.CapacityUnit) + " - " +
		utils.ToTitleCase(preOrder.ContainerType.String()) + " - " +
		utils.ToTitleCase(preOrder.DispenserType.String())

	// Build readable item name (e.g. "Shampoo (250ML - Bottle - Spray)")
	variantName := utils.ToTitleCase(preOrder.ProductVariant.Product.Name) + fmt.Sprintf(" (%s)", variantPropConcat)
	mappedModel := requests.PaymentItemRequest{
		Name:     variantName,
		Quantity: 1,
		Price:    int64(preOrder.UnitPrice),
	}

	items = append(items, mappedModel)
	total += int64(preOrder.UnitPrice)
	return items, total
}

// GetStaffAvailablePreOrdersWithPagination returns preorders for staff with same filtering/search as staff orders
func (s preOrderService) GetStaffAvailablePreOrdersWithPagination(
	limit, page int,
	search, fullName, phone, provinceID, districtID, wardCode string,
	statuses []string,
) ([]responses.PreOrderResponse, int, error) {

	ctx := context.Background()

	// build filter
	filter := func(db *gorm.DB) *gorm.DB {
		q := db

		if search != "" {
			isUUID := false
			if _, err := uuid.Parse(search); err == nil {
				isUUID = true
			}
			if isUUID {
				q = q.Where(`
					(
						pre_orders.id = ?
						OR EXISTS (
							SELECT 1
							FROM payment_transactions pt
							WHERE pt.reference_id = pre_orders.id
							  AND pt.reference_type = ?
							  AND pt.id = ?
						)
					)
				`, search, enum.PaymentTransactionReferenceTypePreOrder, search)
			} else {
				like := "%" + search + "%"
				q = q.Where(`
					(
						pre_orders.id::text ILIKE ?
						OR EXISTS (
							SELECT 1
							FROM payment_transactions pt
							WHERE pt.reference_id = pre_orders.id
							  AND pt.reference_type = ?
							  AND (
									pt.id::text ILIKE ?
									OR pt.payos_metadata->>'bin' ILIKE ?
							  )
						)
					)
				`, like, enum.PaymentTransactionReferenceTypePreOrder, like, like)
			}
		}

		if fullName != "" {
			q = q.Where("full_name ILIKE ?", "%"+fullName+"%")
		}

		if phone != "" {
			q = q.Where("phone_number ILIKE ?", "%"+phone+"%")
		}

		if provinceID != "" {
			q = q.Where("ghn_province_id = ?", provinceID)
		}

		if districtID != "" {
			q = q.Where("ghn_district_id = ?", districtID)
		}

		if wardCode != "" {
			q = q.Where("ghn_ward_code = ?", wardCode)
		}

		// if len(statuses) > 0 {
		// 	q = q.Where("status IN ?", statuses)
		// }
		if len(statuses) > 0 {
			q = q.Where("pre_orders.status IN ?", statuses)
		}

		return q
	}

	includes := []string{
		"ProductVariant.Images",
		"Brand",
		"Category",
	}

	poRows, total, err := s.GetPreOrdersWithPayment(ctx, filter, includes, limit, page)
	if err != nil {
		return nil, 0, err
	}

	// build DTO response
	responsesList := make([]responses.PreOrderResponse, 0, len(poRows))

	for _, r := range poRows {

		var pm *model.PaymentTransaction
		if r.PaymentID != nil {
			pm = &model.PaymentTransaction{
				ID:        *r.PaymentID,
				Amount:    r.PaymentAmount,
				Method:    *r.PaymentMethod,
				Status:    enum.PaymentTransactionStatus(*r.PaymentStatus),
				CreatedAt: *r.PaymentCreatedAt,
			}
		}

		responsesList = append(responsesList,
			responses.PreOrderResponse{}.ToPreOrderResponse(r.PreOrder, pm),
		)
	}

	return responsesList, total, nil
}

type PreOrderWithPayment struct {
	model.PreOrder

	PaymentID        *uuid.UUID `gorm:"column:pm_id"`
	PaymentAmount    *float64   `gorm:"column:pm_amount"`
	PaymentMethod    *string    `gorm:"column:pm_method"`
	PaymentStatus    *string    `gorm:"column:pm_status"`
	PaymentCreatedAt *time.Time `gorm:"column:pm_created_at"`
}

func (s preOrderService) GetPreOrdersWithPayment(
	ctx context.Context,
	filter func(*gorm.DB) *gorm.DB,
	includes []string,
	limit, page int,
) ([]PreOrderWithPayment, int, error) {

	var total int64

	// Step 1: count BEFORE JOIN (avoid inflated count)
	countDB := s.db.WithContext(ctx).Model(&model.PreOrder{})
	if filter != nil {
		countDB = filter(countDB)
	}
	if err := countDB.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Step 2: full JOIN query
	db := s.db.WithContext(ctx).
		Table("pre_orders").
		Select(`
            pre_orders.*,
            pm.id AS pm_id,
            pm.amount AS pm_amount,
            pm.method AS pm_method,
            pm.status AS pm_status,
            pm.created_at AS pm_created_at
        `).
		Joins(`
            LEFT JOIN payment_transactions pm 
            ON pm.reference_id = pre_orders.id 
            AND pm.reference_type = ?
        `, enum.PaymentTransactionReferenceTypePreOrder)

	if filter != nil {
		db = filter(db)
	}

	// Preload relationships (must attach Model)
	db = db.Model(&model.PreOrder{})

	db = db.Order("pre_orders.created_at DESC")

	for _, inc := range includes {
		db = db.Preload(inc)
	}

	if page <= 0 {
		page = 1
	}

	if limit > 0 {
		db = db.Limit(limit).Offset((page - 1) * limit)
	}

	var rows []PreOrderWithPayment
	if err := db.Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, int(total), nil
}

func MovePreOrderStateUsingFSM(preorder *model.PreOrder, lp *model.LimitedProduct, user *model.User, newStatus enum.PreOrderStatus, reason *string) error {
	ctxState := &preordersm.PreOrderContext{
		State:          preordersm.NewPreOrderState(preorder.Status),
		PreOrder:       preorder,
		LimitedProduct: lp,
		ActionBy:       user,
	}
	nextState := preordersm.NewPreOrderState(newStatus)
	if err := ctxState.State.Next(ctxState, nextState); err != nil {
		zap.L().Error("PreOrder state transition validation failed", zap.Error(err))
		return fmt.Errorf("state transition not allowed: %w", err)
	}
	preorder.AddActionNote(*ctxState.GenerateActionNote(user, reason))
	return nil
}

func (p *preOrderService) lookupPreOrderWithLimitedProductAndUser(ctx context.Context, preorderID, actionBy uuid.UUID) (*model.PreOrder, *model.LimitedProduct, *model.User, error) {
	// 1) Load PreOrder
	preOrder, err := p.preOrderRepository.GetByID(ctx, preorderID, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	variantIncludes := []string{"Product", "Product.Limited"}
	variant, err := p.variantRepository.GetByID(ctx, preOrder.VariantID, variantIncludes)
	if err != nil {
		return nil, nil, nil, err
	}

	// If actionBy is zero value, treat as System user
	var user *model.User
	if actionBy == uuid.Nil {
		user = &model.User{
			ID:       uuid.UUID{},
			FullName: p.config.AdminConfig.SystemName,
			Email:    p.config.AdminConfig.SystemEmail,
		}
	} else {
		user, err = p.userRepository.GetByID(ctx, actionBy, nil)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return preOrder, variant.Product.Limited, user, nil
}

func (p *preOrderService) sendNotification(ctx context.Context, notiStatus notiBuilder.PreOrderNotificationType, preOrder *model.PreOrder, actionBy *model.User) error {
	//This contains many payloads to different receivers
	payloads, err := notiBuilder.BuildPreOrderNotifications(ctx, *p.config, p.db, notiStatus, preOrder, actionBy)
	if err != nil {
		zap.L().Debug("no notification builder for order status", zap.Error(err))
		return nil
	}

	for _, pl := range payloads {
		_, err = p.notificationService.CreateAndPublishNotification(ctx, &pl)
		if err != nil {
			zap.L().Error("Failed to send notification", zap.Error(err))
			return err
		}
	}
	return nil
}
