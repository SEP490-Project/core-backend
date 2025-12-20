package irepository

import (
	"context"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type KPIMetricsRepository interface {
	GenericRepository[model.KPIMetrics]
	// GetAggregatedMetrics returns sum of values grouped by type for a reference
	GetAggregatedMetrics(ctx context.Context, referenceID uuid.UUID, referenceType enum.KPIReferenceType, types []enum.KPIValueType) (map[enum.KPIValueType]float64, error)
}
