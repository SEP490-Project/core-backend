package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PreOrderRepository struct {
	*genericRepository[model.PreOrder]
}

func (r *PreOrderRepository) GetStaffAvailablePreOrdersWithPagination(ctx context.Context, limit, page int, search, fullName, phone, provinceID, districtID, wardCode string, statuses []string) ([]model.PreOrder, int, error) {
	pageNum := page
	pageSize := limit
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (pageNum - 1) * pageSize

	// Build base query
	db := r.db.WithContext(ctx).Model(&model.PreOrder{})

	// Validate and filter by statuses
	var validStatuses []string
	for _, s := range statuses {
		s = strings.TrimSpace(s)
		st := enum.PreOrderStatus(s)
		if st.IsValid() {
			validStatuses = append(validStatuses, string(st))
		}
	}

	if len(validStatuses) > 0 {
		db = db.Where("pre_orders.status IN ?", validStatuses)
	}

	// Search by preorder ID or payment transaction ID/bin
	if search != "" {
		isUUID := false
		if _, err := uuid.Parse(search); err == nil {
			isUUID = true
		}

		if isUUID {
			// If search is a valid UUID → exact match on preorder ID or payment transaction ID
			db = db.Where(`(
				pre_orders.id = ? 
				OR EXISTS (
					SELECT 1 FROM payment_transactions pt 
					WHERE pt.reference_id = pre_orders.id 
					AND pt.reference_type = 'PREORDER' 
					AND pt.id = ?
				)
			)`, search, search)
		} else {
			// Otherwise use ILIKE for partial matches
			like := "%" + search + "%"
			db = db.Where(`(
				pre_orders.id::text ILIKE ? 
				OR EXISTS (
					SELECT 1 FROM payment_transactions pt 
					WHERE pt.reference_id = pre_orders.id 
					AND pt.reference_type = 'PREORDER' 
					AND (pt.id::text ILIKE ? OR pt.payos_metadata->>'bin' ILIKE ?)
				)
			)`, like, like, like)
		}
	}

	if fullName != "" {
		db = db.Where("pre_orders.full_name ILIKE ?", "%"+fullName+"%")
	}
	if phone != "" {
		db = db.Where("pre_orders.phone_number ILIKE ?", "%"+phone+"%")
	}
	if provinceID != "" {
		if pid, err := strconv.Atoi(provinceID); err == nil {
			db = db.Where("pre_orders.ghn_province_id = ?", pid)
		}
	}
	if districtID != "" {
		if did, err := strconv.Atoi(districtID); err == nil {
			db = db.Where("pre_orders.ghn_district_id = ?", did)
		}
	}
	if wardCode != "" {
		db = db.Where("pre_orders.ghn_ward_code = ?", wardCode)
	}

	// Count total before pagination
	var total int64
	if err := db.Count(&total).Error; err != nil {
		zap.L().Error("Failed to count staff preorders", zap.Error(err))
		return nil, 0, err
	}

	if total == 0 {
		return []model.PreOrder{}, 0, nil
	}

	// Fetch preorders with includes/preloads
	var preorders []model.PreOrder
	includes := []string{
		"ProductVariant",
		"ProductVariant.Images",
		"ProductVariant.Product",
		"ProductVariant.Product.Limited",
		"Brand",
		"Category",
	}

	queryDB := r.db.WithContext(ctx).Model(&model.PreOrder{})

	// Re-apply filters for the actual query
	if len(validStatuses) > 0 {
		queryDB = queryDB.Where("pre_orders.status IN ?", validStatuses)
	}
	if search != "" {
		isUUID := false
		if _, err := uuid.Parse(search); err == nil {
			isUUID = true
		}
		if isUUID {
			queryDB = queryDB.Where(`(
				pre_orders.id = ? 
				OR EXISTS (
					SELECT 1 FROM payment_transactions pt 
					WHERE pt.reference_id = pre_orders.id 
					AND pt.reference_type = 'PREORDER' 
					AND pt.id = ?
				)
			)`, search, search)
		} else {
			like := "%" + search + "%"
			queryDB = queryDB.Where(`(
				pre_orders.id::text ILIKE ? 
				OR EXISTS (
					SELECT 1 FROM payment_transactions pt 
					WHERE pt.reference_id = pre_orders.id 
					AND pt.reference_type = 'PREORDER' 
					AND (pt.id::text ILIKE ? OR pt.payos_metadata->>'bin' ILIKE ?)
				)
			)`, like, like, like)
		}
	}
	if fullName != "" {
		queryDB = queryDB.Where("pre_orders.full_name ILIKE ?", "%"+fullName+"%")
	}
	if phone != "" {
		queryDB = queryDB.Where("pre_orders.phone_number ILIKE ?", "%"+phone+"%")
	}
	if provinceID != "" {
		if pid, err := strconv.Atoi(provinceID); err == nil {
			queryDB = queryDB.Where("pre_orders.ghn_province_id = ?", pid)
		}
	}
	if districtID != "" {
		if did, err := strconv.Atoi(districtID); err == nil {
			queryDB = queryDB.Where("pre_orders.ghn_district_id = ?", did)
		}
	}
	if wardCode != "" {
		queryDB = queryDB.Where("pre_orders.ghn_ward_code = ?", wardCode)
	}

	// Apply preloads
	for _, inc := range includes {
		queryDB = queryDB.Preload(inc)
	}

	// Order and paginate
	if err := queryDB.Order("pre_orders.created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&preorders).Error; err != nil {
		zap.L().Error("Failed to fetch staff preorders", zap.Error(err))
		return nil, 0, err
	}

	return preorders, int(total), nil
}

func (r *PreOrderRepository) GetPreOrderCountsAndTotalAmountByStatuses(ctx context.Context, statuses []enum.PreOrderStatus, productIDs []uuid.UUID) (count int64, totalAmount float64, err error) {
	db := r.db.WithContext(ctx).Model(&model.PreOrder{})
	query := db.
		Joins("JOIN product_variants pv ON pv.id = pre_orders.variant_id").
		Where("pre_orders.status IN ?", statuses).
		Where("pv.product_id IN ?", productIDs).
		Distinct("pre_orders.id").
		Select("COUNT(pre_orders.id) AS preorder_count, COALESCE(SUM(pre_orders.total_amount), 0) AS total_amount")

	if err = query.Row().Scan(&count, &totalAmount); err != nil {
		zap.L().Error("Failed to get preorder counts and total amount by statuses", zap.Error(err))
		return 0, 0, err
	}
	return count, totalAmount, nil
}

func NewPreOrderRepository(db *gorm.DB) irepository.PreOrderRepository {
	return &PreOrderRepository{genericRepository: &genericRepository[model.PreOrder]{db: db}}
}
