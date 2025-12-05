package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type paymentTransactionRepository struct {
	*genericRepository[model.PaymentTransaction]
}

func NewPaymentTransactionRepository(db *gorm.DB) irepository.PaymentTransactionRepository {
	return &paymentTransactionRepository{
		genericRepository: &genericRepository[model.PaymentTransaction]{db: db},
	}
}

// GetPaymentTransactionByFilter implements irepository.PaymentTransactionRepository.
func (p *paymentTransactionRepository) GetPaymentTransactionByFilter(ctx context.Context, filter *requests.PaymentTransactionFilterRequest) ([]responses.PaymentTransactionResponse, int64, error) {
	db := p.db.WithContext(ctx)
	paginatedQuery := p.applyFilter(db.Model(new(model.PaymentTransaction)), filter)
	countQuery := p.applyFilter(db.Model(new(model.PaymentTransaction)), filter)

	// 1. Count total records
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 2 Apply Sorting
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := (filter.Page - 1) * limit

	sortBy := "created_at"
	sortOrder := "desc"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	if filter.SortOrder != "" && (filter.SortOrder == "asc" || filter.SortOrder == "desc") {
		sortOrder = filter.SortOrder
	}
	paginatedQuery.
		Order(fmt.Sprintf("payment_transactions.%s %s", sortBy, sortOrder)).
		Limit(limit).
		Offset(offset)

	var paymentTransactions []model.PaymentTransaction
	if err := paginatedQuery.Find(&paymentTransactions).Error; err != nil {
		return nil, 0, err
	}
	if len(paymentTransactions) == 0 {
		return nil, 0, nil
	}

	var (
		OrderIDs            []uuid.UUID
		PreOrderIDs         []uuid.UUID
		ContractPaymentIDs  []uuid.UUID
		OrdersMap           = make(map[uuid.UUID]*model.Order)
		PreOrdersMap        = make(map[uuid.UUID]*model.PreOrder)
		ContractPaymentsMap = make(map[uuid.UUID]*model.ContractPayment)
	)

	for _, pt := range paymentTransactions {
		switch pt.ReferenceType {
		case enum.PaymentTransactionReferenceTypeOrder:
			OrderIDs = append(OrderIDs, pt.ReferenceID)

		case enum.PaymentTransactionReferenceTypePreOrder:
			PreOrderIDs = append(PreOrderIDs, pt.ReferenceID)

		case enum.PaymentTransactionReferenceTypeContractPayment:
			ContractPaymentIDs = append(ContractPaymentIDs, pt.ReferenceID)

		}
	}
<<<<<<< HEAD
	OrdersMap, _ = p.GetReferenceOrderByIDs(ctx, OrderIDs)
	PreOrdersMap, _ = p.GetReferencePreOrderByIDs(ctx, PreOrderIDs)
	ContractPaymentsMap, _ = p.GetReferenceContractPaymentByIDs(ctx, ContractPaymentIDs)
=======

	if len(OrderIDs) > 0 {
		var orderInfo []model.Order
		orderQuery := p.db.WithContext(ctx).
			Model(new(model.Order)).
			Preload("OrderItems").
			Select(
				"orders.id",
				"orders.user_id",
				"orders.full_name",
				"orders.phone_number",
				"orders.email",
				"orders.user_bank_account",
				"orders.user_bank_name",
				"orders.user_bank_account_holder",
			).
			Where("orders.id IN ?", OrderIDs)
		if err := orderQuery.Find(&orderInfo).Error; err != nil {
			return nil, 0, err
		}
		for _, order := range orderInfo {
			OrdersMap[order.ID] = &order
		}
	}
	if len(PreOrderIDs) > 0 {
		var preOrderInfo []model.PreOrder
		preOrderQuery := p.db.WithContext(ctx).
			Model(new(model.PreOrder)).
			Select(
				"pre_orders.id",
				"pre_orders.user_id",
				"pre_orders.full_name",
				"pre_orders.phone_number",
				"pre_orders.email",
				"pre_orders.user_bank_account",
				"pre_orders.user_bank_name",
				"pre_orders.user_bank_account_holder",
				"pre_orders.variant_id",
				"pre_orders.product_name",
				"pre_orders.quantity",
				"pre_orders.unit_price",
				"pre_orders.total_amount",
			).
			Where("pre_orders.id IN ?", PreOrderIDs)
		if err := preOrderQuery.Find(&preOrderInfo).Error; err != nil {
			return nil, 0, err
		}
		for _, preOrder := range preOrderInfo {
			PreOrdersMap[preOrder.ID] = &preOrder
		}
	}

	if len(ContractPaymentIDs) > 0 {
		var contractPaymentInfo []model.ContractPayment
		contractPaymentQuery := p.db.WithContext(ctx).
			Model(new(model.ContractPayment)).
			Joins("Contract").
			Joins("Contract.Brand").
			Select(
				"contract_payments.id",
				"contract_payments.contract_id",
				"contract_payments.is_deposit",
			).
			Where("contract_payments.id IN ?", ContractPaymentIDs)
		if err := contractPaymentQuery.Find(&contractPaymentInfo).Error; err != nil {
			return nil, 0, err
		}
		for _, contractPayment := range contractPaymentInfo {
			ContractPaymentsMap[contractPayment.ID] = &contractPayment
		}
	}
>>>>>>> 9a816cd (Feat/transaction reference info 20251203 (#200))

	paymentResponses := make([]responses.PaymentTransactionResponse, len(paymentTransactions))
	for i, pt := range paymentTransactions {
		var additionalInfo any
		switch pt.ReferenceType {
		case enum.PaymentTransactionReferenceTypeOrder:
<<<<<<< HEAD
			if order, ok := OrdersMap[pt.ReferenceID]; ok && order != nil {
				additionalInfo = responses.PaymentTransactionReferenceOrder{}.FromOrderModel(order)
			}
		case enum.PaymentTransactionReferenceTypePreOrder:
			if preOrder, ok := PreOrdersMap[pt.ReferenceID]; ok && preOrder != nil {
				additionalInfo = responses.PaymentTransactionReferencePreOrder{}.FromPreOrderModel(preOrder)
			}
		case enum.PaymentTransactionReferenceTypeContractPayment:
			if contractPayment, ok := ContractPaymentsMap[pt.ReferenceID]; ok && ContractPaymentsMap[pt.ReferenceID] != nil {
				additionalInfo = responses.PaymentTransactionReferenceContractPayment{}.FromContractPaymentModel(contractPayment)
			}
=======
			additionalInfo = responses.PaymentTransactionReferenceOrder{}.FromOrderModel(OrdersMap[pt.ReferenceID])
		case enum.PaymentTransactionReferenceTypePreOrder:
			additionalInfo = responses.PaymentTransactionReferencePreOrder{}.FromPreOrderModel(PreOrdersMap[pt.ReferenceID])
		case enum.PaymentTransactionReferenceTypeContractPayment:
			additionalInfo = responses.PaymentTransactionReferenceContractPayment{}.FromContractPaymentModel(ContractPaymentsMap[pt.ReferenceID])
>>>>>>> 9a816cd (Feat/transaction reference info 20251203 (#200))
		}

		paymentResponses[i] = *responses.PaymentTransactionResponse{}.ToResponse(&pt, additionalInfo)
	}
	return paymentResponses, total, nil
}

// GetPaymentTransactionByID implements irepository.PaymentTransactionRepository.
func (p *paymentTransactionRepository) GetPaymentTransactionByID(ctx context.Context, ID uuid.UUID) (*responses.PaymentTransactionResponse, error) {
<<<<<<< HEAD
	db := p.db.WithContext(ctx)
	var paymentTransaction model.PaymentTransaction

	if err := db.Model(new(model.PaymentTransaction)).
		Where("payment_transactions.id = ?", ID).
		First(&paymentTransaction).Error; err != nil {
		return nil, err
	}
	var additionalInfo any
	switch paymentTransaction.ReferenceType {
	case enum.PaymentTransactionReferenceTypeOrder:
		if order, err := p.GetReferenceOrderByIDs(ctx, []uuid.UUID{paymentTransaction.ReferenceID}); err == nil {
			if orderInfo, ok := order[paymentTransaction.ReferenceID]; ok && orderInfo != nil {
				additionalInfo = responses.PaymentTransactionReferenceOrder{}.FromOrderModel(orderInfo)
			}
		}

	case enum.PaymentTransactionReferenceTypePreOrder:
		if preOrder, err := p.GetReferencePreOrderByIDs(ctx, []uuid.UUID{paymentTransaction.ReferenceID}); err == nil {
			if preOrderInfo, ok := preOrder[paymentTransaction.ReferenceID]; ok && preOrderInfo != nil {
				additionalInfo = responses.PaymentTransactionReferencePreOrder{}.FromPreOrderModel(preOrderInfo)
			}
		}

	case enum.PaymentTransactionReferenceTypeContractPayment:
		if contractPayment, err := p.GetReferenceContractPaymentByIDs(ctx, []uuid.UUID{paymentTransaction.ReferenceID}); err == nil {
			if contractPaymentInfo, ok := contractPayment[paymentTransaction.ReferenceID]; ok && contractPaymentInfo != nil {
				additionalInfo = responses.PaymentTransactionReferenceContractPayment{}.FromContractPaymentModel(contractPaymentInfo)
			}
		}
	}

	paymentResponse := *responses.PaymentTransactionResponse{}.ToResponse(&paymentTransaction, additionalInfo)
	return &paymentResponse, nil
=======
	panic("unimplemented")
>>>>>>> 9a816cd (Feat/transaction reference info 20251203 (#200))
}

func (p *paymentTransactionRepository) applyFilter(query *gorm.DB, filter *requests.PaymentTransactionFilterRequest) *gorm.DB {
	if filter.OrderCode != nil {
		query = query.Where("payment_transactions.payos_metadata->>'order_code' = ?", strconv.FormatInt(int64(*filter.OrderCode), 10))
	}
	if filter.ReferenceID != nil {
		query = query.Where("payment_transactions.reference_id = ?", filter.ReferenceID)
	}
	if filter.ReferenceType != nil {
		query = query.Where("payment_transactions.reference_type = ?", filter.ReferenceType.String())
	}
	if filter.PayerID != nil {
		query = query.Where("payment_transactions.payer_id = ?", filter.PayerID)
	}
	if filter.Status != nil {
		query = query.Where("payment_transactions.status = ?", filter.Status.String())
	}
	if filter.TransactionFromDate != nil {
		fromDate := utils.ParseLocalTimeWithFallback(*filter.TransactionFromDate, utils.DateFormat)
		if fromDate != nil {
			query = query.Where("payment_transactions.transaction_date >= ?", fromDate)
		}
	}
	if filter.TransactionToDate != nil {
		toDate := utils.ParseLocalTimeWithFallback(*filter.TransactionToDate, utils.DateFormat)
		if toDate != nil {
			query = query.Where("payment_transactions.transaction_date <= ?", toDate)
		}
	}
	if filter.TransactionToDate != nil {
		toDate := utils.ParseLocalTimeWithFallback(*filter.TransactionToDate, utils.DateFormat)
		if toDate != nil {
			query = query.Where("payment_transactions.transaction_date <= ?", toDate)
		}
	}

	return query
}
<<<<<<< HEAD

func (p *paymentTransactionRepository) GetReferenceOrderByIDs(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID]*model.Order, error) {
	var ordersMap = make(map[uuid.UUID]*model.Order)
	if len(orderIDs) == 0 {
		return ordersMap, nil
	}

	var orderInfo []model.Order
	orderQuery := p.db.WithContext(ctx).
		Model(new(model.Order)).
		Preload("OrderItems").
		Select(
			"orders.id",
			"orders.user_id",
			"orders.full_name",
			"orders.phone_number",
			"orders.email",
			"orders.user_bank_account",
			"orders.user_bank_name",
			"orders.user_bank_account_holder",
		).
		Where("orders.id IN ?", orderIDs)
	if err := orderQuery.Find(&orderInfo).Error; err != nil {
		return ordersMap, err
	}
	for _, order := range orderInfo {
		ordersMap[order.ID] = &order
	}
	return ordersMap, nil
}

func (p *paymentTransactionRepository) GetReferencePreOrderByIDs(ctx context.Context, preOrderIDs []uuid.UUID) (map[uuid.UUID]*model.PreOrder, error) {
	var preOrdersMap = make(map[uuid.UUID]*model.PreOrder)
	if len(preOrderIDs) == 0 {
		return preOrdersMap, nil
	}

	var preOrderInfo []model.PreOrder
	preOrderQuery := p.db.WithContext(ctx).
		Model(new(model.PreOrder)).
		Select(
			"pre_orders.id",
			"pre_orders.user_id",
			"pre_orders.full_name",
			"pre_orders.phone_number",
			"pre_orders.email",
			"pre_orders.user_bank_account",
			"pre_orders.user_bank_name",
			"pre_orders.user_bank_account_holder",
			"pre_orders.variant_id",
			"pre_orders.product_name",
			"pre_orders.quantity",
			"pre_orders.unit_price",
			"pre_orders.total_amount",
		).
		Where("pre_orders.id IN ?", preOrderIDs)
	if err := preOrderQuery.Find(&preOrderInfo).Error; err != nil {
		return preOrdersMap, err
	}
	for _, preOrder := range preOrderInfo {
		preOrdersMap[preOrder.ID] = &preOrder
	}
	return preOrdersMap, nil
}

func (p *paymentTransactionRepository) GetReferenceContractPaymentByIDs(ctx context.Context, contractPaymentIDs []uuid.UUID) (map[uuid.UUID]*model.ContractPayment, error) {
	var contractPaymentsMap = make(map[uuid.UUID]*model.ContractPayment)
	if len(contractPaymentIDs) == 0 {
		return contractPaymentsMap, nil
	}

	var contractPaymentInfo []model.ContractPayment
	contractPaymentQuery := p.db.WithContext(ctx).
		Model(new(model.ContractPayment)).
		Joins("Contract").
		Joins("Contract.Brand").
		Select(
			"contract_payments.id",
			"contract_payments.contract_id",
			"contract_payments.is_deposit",
		).
		Where("contract_payments.id IN ?", contractPaymentIDs)
	if err := contractPaymentQuery.Find(&contractPaymentInfo).Error; err != nil {
		return contractPaymentsMap, err
	}
	for _, contractPayment := range contractPaymentInfo {
		contractPaymentsMap[contractPayment.ID] = &contractPayment
	}
	return contractPaymentsMap, nil
}
=======
>>>>>>> 9a816cd (Feat/transaction reference info 20251203 (#200))
