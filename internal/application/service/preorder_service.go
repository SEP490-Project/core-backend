package service

import (
	"context"
	"core-backend/config"
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
	"fmt"
	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
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

		//validate variant and stocks/products
		variant, err := uow.ProductVariant().GetByID(ctx, request.VariantID, []string{"Product"})
		if err != nil {
			return fmt.Errorf("variant %w not found", err)
		} else if err := validateVariantForPreOrder(*variant); err != nil {
			return err
		}

		//validate shipping address
		address, err := uow.ShippingAddresses().GetByID(ctx, request.AddressID, []string{})
		if err != nil {
			return fmt.Errorf("failed to get shipping address: %w", err)
		}

		preOrder := requests.PreOrderRequest{}.ToModel(*address, *variant, now)
		//minus 1 stock
		variant.CurrentStock = ptr.Int(*variant.CurrentStock - 1)
		err = uow.ProductVariant().Update(ctx, variant)
		if err != nil {
			return fmt.Errorf("failed to update variant stock: %w", err)
		}
		//create pre-order
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

func (p preOrderService) GetPreOrdersByUserIDWithPagination(userID uuid.UUID, limit, page int, search string) ([]model.Order, int, error) {
	//TODO implement me
	panic("implement me")
}

func (p preOrderService) PayPreOrder(ctx context.Context, orderID uuid.UUID, shippingPrice int, successURL, cancelURL string, unitOfWork irepository.UnitOfWork) (*model.PaymentTransaction, error) {
	//TODO implement me
	panic("implement me")
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

// Validator
func validateVariantForPreOrder(variant model.ProductVariant) error {
	//validate if product type is limited
	if variant.Product != nil && variant.Product.Type != enum.ProductTypeLimited {
		return fmt.Errorf("invalid product type for pre-order")
	}

	//validate if product is active
	if variant.Product.IsActive == false {
		return fmt.Errorf("product is not available")
	}

	//Check stock
	if *variant.CurrentStock == 0 {
		return fmt.Errorf("variant is out of stock")
	}
	return nil
}
