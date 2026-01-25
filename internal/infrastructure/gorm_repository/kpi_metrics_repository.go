package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type kpiMetricsRepository struct {
	irepository.GenericRepository[model.KPIMetrics]
	db *gorm.DB
}

func NewKPIMetricsRepository(db *gorm.DB) *kpiMetricsRepository {
	return &kpiMetricsRepository{
		GenericRepository: NewGenericRepository[model.KPIMetrics](db),
		db:                db,
	}
}

func (r *kpiMetricsRepository) GetAggregatedMetrics(ctx context.Context, referenceID uuid.UUID, referenceType enum.KPIReferenceType, types []enum.KPIValueType) (map[enum.KPIValueType]float64, error) {
	type Result struct {
		Type  enum.KPIValueType
		Total float64
	}
	var results []Result

	query := r.db.WithContext(ctx).
		Model(&model.KPIMetrics{}).
		Where("reference_id = ? AND reference_type = ?", referenceID, referenceType)

	if len(types) > 0 {
		query = query.Where("type IN ?", types)
	}

	err := query.Select("type, COALESCE(SUM(value), 0) as total").
		Group("type").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	metrics := make(map[enum.KPIValueType]float64)
	for _, res := range results {
		metrics[res.Type] = res.Total
	}
	return metrics, nil
}

func (r *kpiMetricsRepository) GetAggregatedMetricsByReferenceIDs(ctx context.Context, referenceIDs []uuid.UUID, referenceType enum.KPIReferenceType, types []enum.KPIValueType) (map[enum.KPIValueType]float64, error) {
	if len(referenceIDs) == 0 || referenceType == "" {
		panic("referenceIDs and referenceType must be provided")
	}

	type Result struct {
		Type  enum.KPIValueType
		Total float64
	}
	var results []Result
	query := r.db.WithContext(ctx).
		Model(&model.KPIMetrics{}).
		Where("reference_id IN ? AND reference_type = ?", referenceIDs, referenceType)

	if len(types) > 0 {
		query = query.Where("type IN ?", types)
	}

	if err := query.
		Select("type, COALESCE(SUM(value), 0) as total").
		Group("type").
		Scan(&results).
		Error; err != nil {
		return nil, err
	}

	metrics := make(map[enum.KPIValueType]float64)
	for _, res := range results {
		metrics[res.Type] = res.Total
	}
	return metrics, nil
}
