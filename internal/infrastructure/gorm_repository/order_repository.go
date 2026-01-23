package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OrderRepository struct {
	*genericRepository[model.Order]
}

func (r *OrderRepository) GetStaffAvailableOrdersWithPagination(ctx context.Context, limit, page int, search, fullName, phone, provinceID, districtID, wardCode, orderType, createdFrom, createdTo, brandID string, statuses []string) ([]model.Order, int, error) {
	pageNum := page
	pageSize := limit
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (pageNum - 1) * pageSize

	var validStatuses []string
	for _, s := range statuses {
		s = strings.TrimSpace(s)
		// if s == "" || s == string(enum.OrderStatusPending) {
		// 	continue
		// }
		st := enum.OrderStatus(s)
		if st.IsValid() {
			validStatuses = append(validStatuses, string(st))
		}
	}

	whereClauses := make([]string, 0)
	args := make([]any, 0)

	if len(validStatuses) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("orders.status IN (%s)", strings.Repeat("?,", len(validStatuses)-1)+"?"))
		for _, v := range validStatuses {
			args = append(args, v)
		}
	}

	// exclude PENDING
	whereClauses = append(whereClauses, "orders.status <> ?")
	args = append(args, enum.OrderStatusPending)

	if search != "" {
		isUUID := false
		if _, err := uuid.Parse(search); err == nil {
			isUUID = true
		}

		if isUUID {
			// Nếu search là UUID thật → so sánh chính xác
			whereClauses = append(whereClauses, "(orders.id = ? OR pt_latest.id = ?)")
			args = append(args, search, search)
		} else {
			// Nếu không phải UUID → dùng ILIKE như cũ
			whereClauses = append(whereClauses, "(orders.id::text ILIKE ? OR pt_latest.id::text ILIKE ? OR pt_latest.payos_metadata->>'bin' ILIKE ?)")
			like := "%" + search + "%"
			args = append(args, like, like, like)
		}
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
	if orderType != "" {
		ot := enum.ProductType(orderType)
		if ot.IsValid() {
			whereClauses = append(whereClauses, "orders.order_type = ?")
			args = append(args, ot)
		}
	}
	if createdFrom != "" {
		whereClauses = append(whereClauses, "orders.created_at >= ?")
		args = append(args, createdFrom)
	}
	if createdTo != "" {
		whereClauses = append(whereClauses, "orders.created_at <= ?")
		args = append(args, createdTo+" 23:59:59")
	}
	if brandID != "" {
		if _, err := uuid.Parse(brandID); err == nil {
			whereClauses = append(whereClauses, "EXISTS (SELECT 1 FROM order_items oi WHERE oi.order_id = orders.id AND oi.brand_id = ?)")
			args = append(args, brandID)
		}
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	type orderWithTotal struct {
		model.Order
		PaymentID  *uuid.UUID `gorm:"column:payment_id"`
		Bin        *string    `gorm:"column:bin"`
		TotalCount int64      `gorm:"column:total_count"`
	}

	// Join only the latest payment transaction per order using window function
	// We select latest by updated_at DESC and keep rn = 1
	sql := fmt.Sprintf(`SELECT orders.*, pt_latest.id AS payment_id, pt_latest.payos_metadata->>'bin' AS bin, COUNT(*) OVER() AS total_count
        FROM orders
        LEFT JOIN (
            SELECT id, reference_id, payos_metadata, updated_at, ROW_NUMBER() OVER (PARTITION BY reference_id ORDER BY updated_at DESC) AS rn
            FROM payment_transactions
            WHERE reference_type = 'ORDER'
        ) pt_latest ON pt_latest.reference_id = orders.id AND pt_latest.rn = 1
        WHERE %s
        ORDER BY orders.created_at DESC
        LIMIT ? OFFSET ?`, whereSQL)
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
	if len(orderIDs) == 0 {
		// no order items to load
		return orders, total, nil
	}

	var items []model.OrderItem
	if err := r.db.WithContext(ctx).
		Model(&model.OrderItem{}).
		Preload("Brand").
		Preload("Category").
		Preload("Variant").
		Preload("Variant.Images").
		Preload("Variant.Product").
		Preload("Variant.Product.Limited").
		Where("order_id IN ?", orderIDs).
		Find(&items).Error; err != nil {
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

func (r *OrderRepository) GetSelfDeliveryOrdersWithPagination(ctx context.Context, limit, page int, search, status, fullName, phone, provinceID, districtID, wardCode string) ([]model.Order, int, error) {
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

	// filter is_self_picked_up = false
	whereClauses = append(whereClauses, "orders.is_self_picked_up = ?")
	args = append(args, false)

	//filter order_type = 'LIMITED'
	whereClauses = append(whereClauses, "orders.order_type = ?")
	args = append(args, enum.ProductTypeLimited)

	if validStatus != nil && *validStatus != enum.OrderStatusPending {
		whereClauses = append(whereClauses, "orders.status = ?")
		args = append(args, *validStatus)
	}
	if search != "" {
		isUUID := false
		if _, err := uuid.Parse(search); err == nil {
			isUUID = true
		}

		if isUUID {
			// Nếu search là UUID thật → so sánh chính xác
			whereClauses = append(whereClauses, "(orders.id = ? OR pt_latest.id = ?)")
			args = append(args, search, search)
		} else {
			// Nếu không phải UUID → dùng ILIKE như cũ
			whereClauses = append(whereClauses, "(orders.id::text ILIKE ? OR pt_latest.id::text ILIKE ? OR pt_latest.payos_metadata->>'bin' ILIKE ?)")
			like := "%" + search + "%"
			args = append(args, like, like, like)
		}
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

	// Join only the latest payment transaction per order using window function
	sql := fmt.Sprintf(`SELECT orders.*, pt_latest.id AS payment_id, pt_latest.payos_metadata->>'bin' AS bin, COUNT(*) OVER() AS total_count
        FROM orders
        LEFT JOIN (
            SELECT id, reference_id, payos_metadata, updated_at, ROW_NUMBER() OVER (PARTITION BY reference_id ORDER BY updated_at DESC) AS rn
            FROM payment_transactions
            WHERE reference_type = 'ORDER'
        ) pt_latest ON pt_latest.reference_id = orders.id AND pt_latest.rn = 1
        WHERE %s
        ORDER BY orders.created_at DESC
        LIMIT ? OFFSET ?`, whereSQL)
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

// GetOrdersWithFiltersWithPagination searches orders by GHN order code or Order ID,
// filters by created date range and by status. It follows same pagination pattern
// as other repository methods and includes payment join for additional metadata.
func (r *OrderRepository) GetOrdersWithFiltersWithPagination(
	ctx context.Context,
	limit, page int,
	search, status, createdFrom, createdTo, fullName, phone, provinceID, districtID, wardCode, orderType string,
) ([]model.Order, int, error) {
	pageNum := page
	pageSize := limit
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (pageNum - 1) * pageSize

	// Build WHERE clauses similar to other methods so we can reuse the raw SQL + window-join
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

	if createdFrom != "" {
		if t, err := time.Parse("2006-01-02", createdFrom); err == nil {
			whereClauses = append(whereClauses, "orders.created_at >= ?")
			args = append(args, t)
		}
	}
	if createdTo != "" {
		if t, err := time.Parse("2006-01-02", createdTo); err == nil {
			whereClauses = append(whereClauses, "orders.created_at < ?")
			args = append(args, t.Add(24*time.Hour))
		}
	}

	if search != "" {
		if id, err := uuid.Parse(search); err == nil {
			whereClauses = append(whereClauses, "(orders.id = ? OR orders.ghn_order_code = ?)")
			args = append(args, id, search)
		} else {
			like := "%" + search + "%"
			whereClauses = append(whereClauses, "(orders.ghn_order_code ILIKE ? OR orders.id::text ILIKE ?)")
			args = append(args, like, like)
		}
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
	if orderType != "" {
		ot := enum.ProductType(orderType)
		if ot.IsValid() {
			whereClauses = append(whereClauses, "orders.order_type = ?")
			args = append(args, ot)
		}
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	type orderWithTotal struct {
		model.Order
		PaymentID  *uuid.UUID `gorm:"column:payment_id"`
		PaymentBin *string    `gorm:"column:payment_bin"`
		TotalCount int64      `gorm:"column:total_count"`
	}

	// Use same window-function subquery to pick latest payment transaction per order
	sql := fmt.Sprintf(`SELECT orders.*, pt_latest.id AS payment_id, pt_latest.payos_metadata->>'bin' AS payment_bin, COUNT(*) OVER() AS total_count
		FROM orders
		LEFT JOIN (
			SELECT id, reference_id, payos_metadata, updated_at, ROW_NUMBER() OVER (PARTITION BY reference_id ORDER BY updated_at DESC) AS rn
			FROM payment_transactions
			WHERE reference_type = 'ORDER'
		) pt_latest ON pt_latest.reference_id = orders.id AND pt_latest.rn = 1
		WHERE %s
		ORDER BY orders.created_at DESC
		LIMIT ? OFFSET ?`, whereSQL)
	args = append(args, pageSize, offset)

	var rows []orderWithTotal
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		zap.L().Error("Failed to execute orders-with-filters raw query", zap.Error(err))
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
			binMap[r2.ID] = r2.PaymentBin
		} else {
			paymentMap[r2.ID] = nil
			binMap[r2.ID] = nil
		}
	}

	// load items in bulk
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
		if pid := paymentMap[orders[i].ID]; pid != nil {
			orders[i].PaymentID = pid
		}
		if b := binMap[orders[i].ID]; b != nil {
			orders[i].PaymentBin = b
		}
	}

	return orders, total, nil
}

func (r *OrderRepository) GetOrderCountsAndTotalRevenueByOrderType(
	ctx context.Context, orderType enum.ProductType, status []enum.OrderStatus, productIDs []uuid.UUID,
) (count int64, totalRevenue float64, err error) {
	db := r.db.WithContext(ctx).Model(&model.Order{})
	query := db.
		Joins("JOIN order_items oi ON oi.order_id = orders.id").
		Joins("JOIN product_variants pv ON pv.id = oi.variant_id").
		Where("orders.order_type = ?", orderType).
		Where("orders.status IN ?", status).
		Where("pv.product_id IN ?", productIDs).
		Distinct("orders.id").
		Select("COUNT(orders.id) AS order_count, COALESCE(SUM(orders.total_amount), 0) AS total_revenue")

	if err = query.Row().Scan(&count, &totalRevenue); err != nil {
		zap.L().Error("Failed to get order counts and total revenue by order type", zap.Error(err))
		return 0, 0, err
	}
	return count, totalRevenue, nil
}

func NewOrderRepository(db *gorm.DB) irepository.OrderRepository {
	return &OrderRepository{genericRepository: &genericRepository[model.Order]{db: db}}
}
