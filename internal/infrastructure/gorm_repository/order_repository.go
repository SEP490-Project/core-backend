package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OrderRepository struct {
	*genericRepository[model.Order]
}

func (r *OrderRepository) GetStaffAvailableOrdersWithPagination(ctx context.Context, limit, page int, search, status, fullName, phone, provinceID, districtID, wardCode string) ([]model.Order, int, error) {
	pageNum := page
	pageSize := limit
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (pageNum - 1) * pageSize

	var validStatus *enum.OrderStatus
	if status != "" {
		s := enum.OrderStatus(status)
		if s.IsValid() {
			validStatus = &s
		}
	}

	whereClauses := make([]string, 0)
	args := make([]any, 0)

	// exclude PENDING
	whereClauses = append(whereClauses, "orders.status <> ?")
	args = append(args, enum.OrderStatusPending)

	if validStatus != nil && *validStatus != enum.OrderStatusPending {
		whereClauses = append(whereClauses, "orders.status = ?")
		args = append(args, *validStatus)
	}
	if search != "" {
		// Search across order number, payment transaction id, and payos_metadata.bin
		whereClauses = append(whereClauses, "(orders.id ILIKE ? OR pt.id::text ILIKE ? OR pt.payos_metadata->>'bin' ILIKE ?)")
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	if fullName != "" {
		whereClauses = append(whereClauses, "orders.full_name ILIKE ?")
		args = append(args, "%"+fullName+"%")
	}
	if phone != "" {
		whereClauses = append(whereClauses, "orders.phone_number ILIKE ?")
		args = append(args, "%"+phone+"%")
	}
	if provinceID != "" {
		if pid, err := strconv.Atoi(provinceID); err == nil {
			whereClauses = append(whereClauses, "orders.ghn_province_id = ?")
			args = append(args, pid)
		}
	}
	if districtID != "" {
		if did, err := strconv.Atoi(districtID); err == nil {
			whereClauses = append(whereClauses, "orders.ghn_district_id = ?")
			args = append(args, did)
		}
	}
	if wardCode != "" {
		whereClauses = append(whereClauses, "orders.ghn_ward_code = ?")
		args = append(args, wardCode)
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	type orderWithTotal struct {
		model.Order
		PaymentID  *uuid.UUID `gorm:"column:payment_id"`
		Bin        *string    `gorm:"column:bin"`
		TotalCount int64      `gorm:"column:total_count"`
	}

	// Join payment_transactions (left join so orders without payments are included)
	sql := fmt.Sprintf(`SELECT orders.*, pt.id AS payment_id, pt.payos_metadata->>'bin' AS bin, COUNT(*) OVER() AS total_count FROM orders LEFT JOIN payment_transactions pt ON pt.reference_id = orders.id AND pt.reference_type = 'ORDER' WHERE %s ORDER BY orders.created_at DESC LIMIT ? OFFSET ?`, whereSQL)
	args = append(args, pageSize, offset)

	var rows []orderWithTotal
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		zap.L().Error("Failed to execute staff orders raw query", zap.Error(err))
		return nil, 0, err
	}

	if len(rows) == 0 {
		return []model.Order{}, 0, nil
	}

	total := int(rows[0].TotalCount)
	orders := make([]model.Order, 0, len(rows))
	orderIDs := make([]uuid.UUID, 0, len(rows))
	paymentMap := make(map[uuid.UUID]*uuid.UUID)
	binMap := make(map[uuid.UUID]*string)
	for _, r2 := range rows {
		orders = append(orders, r2.Order)
		orderIDs = append(orderIDs, r2.ID)
		if r2.PaymentID != nil {
			paymentMap[r2.ID] = r2.PaymentID
			binMap[r2.ID] = r2.Bin
		} else {
			paymentMap[r2.ID] = nil
			binMap[r2.ID] = nil
		}
	}

	// load items
	var items []model.OrderItem
	if err := r.db.WithContext(ctx).Model(&model.OrderItem{}).Where("order_id IN ?", orderIDs).Find(&items).Error; err != nil {
		zap.L().Error("Failed to load order items for orders", zap.Error(err))
		return nil, 0, err
	}

	itemMap := make(map[uuid.UUID][]model.OrderItem)
	for _, it := range items {
		itemMap[it.OrderID] = append(itemMap[it.OrderID], it)
	}

	for i := range orders {
		orders[i].OrderItems = itemMap[orders[i].ID]
		// attach payment info into transient fields on Order (gorm:"-" so not persisted)
		if pid := paymentMap[orders[i].ID]; pid != nil {
			orders[i].PaymentID = pid
		}
		if b := binMap[orders[i].ID]; b != nil {
			orders[i].PaymentBin = b
		}
	}

	return orders, total, nil
}

func NewOrderRepository(db *gorm.DB) irepository.OrderRepository {
	return &OrderRepository{genericRepository: &genericRepository[model.Order]{db: db}}
}
