package iservice

import (
	"context"
	"core-backend/internal/application/dto/responses"
)

type SystemService interface {
	GetSystemSpecs(ctx context.Context) (*responses.SystemSpecsResponse, error)
}
