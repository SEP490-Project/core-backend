package handler

import (
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"

	"github.com/go-playground/validator/v10"
)

type ModifiedHistoryHandler struct {
	modifiedHistoryService iservice.ModifiedHistoryService
	unitOfWork             irepository.UnitOfWork
	validator              *validator.Validate
}

func NewModifiedHistoryHandler(
	modifiedHistoryService iservice.ModifiedHistoryService,
	unitOfWork irepository.UnitOfWork,
) *ModifiedHistoryHandler {
	validator := validator.New()
	return &ModifiedHistoryHandler{
		modifiedHistoryService: modifiedHistoryService,
		unitOfWork:             unitOfWork,
		validator:              validator,
	}
}
