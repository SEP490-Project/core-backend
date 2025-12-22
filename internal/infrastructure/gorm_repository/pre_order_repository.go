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

type PreOrderRepository struct {
	*genericRepository[model.PreOrder]
}

func (r *PreOrderRepository) GetStaffAvailablePreOrdersWithPagination(ctx context.Context, limit, page int, search, status, fullName, phone, provinceID, districtID, wardCode string) ([]model.PreOrder, int, error) {
	pageNum := page
	pageSize := limit
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (pageNum - 1) * pageSize

	var validStatus *enum.PreOrderStatus
	if status != "" {
		s := enum.PreOrderStatus(status)
		if s.IsValid() {
			validStatus = &s
		}
	}

	whereClauses := make([]string, 0)
	args := make([]any, 0)

	// exclude PENDING by default to match staff view behavior
	// whereClauses = append(whereClauses, "pre_orders.status <> ?")
	// args = append(args, enum.PreOrderStatusPending)

	// if validStatus != nil && *validStatus != enum.PreOrderStatusPending {
	// 	whereClauses = append(whereClauses, "pre_orders.status = ?")
	// 	args = append(args, *validStatus)
	// }

	if validStatus != nil {
		whereClauses = append(whereClauses, "pre_orders.status = ?")
		args = append(args, *validStatus)
	}

	if search != "" {
		whereClauses = append(whereClauses, "(pre_orders.id::text ILIKE ? OR pt.id::text ILIKE ? OR pt.payos_metadata->>'bin' ILIKE ?)")
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	if fullName != "" {
		whereClauses = append(whereClauses, "pre_orders.full_name ILIKE ?")
		args = append(args, "%"+fullName+"%")
	}
	if phone != "" {
		whereClauses = append(whereClauses, "pre_orders.phone_number ILIKE ?")
		args = append(args, "%"+phone+"%")
	}
	if provinceID != "" {
		if pid, err := strconv.Atoi(provinceID); err == nil {
			whereClauses = append(whereClauses, "pre_orders.ghn_province_id = ?")
			args = append(args, pid)
		}
	}
	if districtID != "" {
		if did, err := strconv.Atoi(districtID); err == nil {
			whereClauses = append(whereClauses, "pre_orders.ghn_district_id = ?")
			args = append(args, did)
		}
	}
	if wardCode != "" {
		whereClauses = append(whereClauses, "pre_orders.ghn_ward_code = ?")
		args = append(args, wardCode)
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	type preOrderWithTotal struct {
		model.PreOrder
		PaymentID  *uuid.UUID `gorm:"column:payment_id"`
		Bin        *string    `gorm:"column:bin"`
		TotalCount int64      `gorm:"column:total_count"`
	}

	sql := fmt.Sprintf(`SELECT pre_orders.*, pt.id AS payment_id, pt.payos_metadata->>'bin' AS bin, COUNT(*) OVER() AS total_count FROM pre_orders LEFT JOIN payment_transactions pt ON pt.reference_id = pre_orders.id AND pt.reference_type = 'PREORDER' WHERE %s ORDER BY pre_orders.created_at DESC LIMIT ? OFFSET ?`, whereSQL)
	args = append(args, pageSize, offset)

	var rows []preOrderWithTotal
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		zap.L().Error("Failed to execute staff preorders raw query", zap.Error(err))
		return nil, 0, err
	}

	if len(rows) == 0 {
		return []model.PreOrder{}, 0, nil
	}

	total := int(rows[0].TotalCount)
	preorders := make([]model.PreOrder, 0, len(rows))
	preorderIDs := make([]uuid.UUID, 0, len(rows))
	paymentMap := make(map[uuid.UUID]*uuid.UUID)
	binMap := make(map[uuid.UUID]*string)
	for _, r2 := range rows {
		preorders = append(preorders, r2.PreOrder)
		preorderIDs = append(preorderIDs, r2.ID)
		if r2.PaymentID != nil {
			paymentMap[r2.ID] = r2.PaymentID
			binMap[r2.ID] = r2.Bin
		} else {
			paymentMap[r2.ID] = nil
			binMap[r2.ID] = nil
		}
	}

	// Attach transient payment fields
	for i := range preorders {
		if pid := paymentMap[preorders[i].ID]; pid != nil {
			preorders[i].PaymentID = pid
		}
		if b := binMap[preorders[i].ID]; b != nil {
			preorders[i].PaymentBin = b
		}
	}

	return preorders, total, nil
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
