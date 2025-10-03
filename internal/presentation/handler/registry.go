// Package handler implements HTTP handlers for the application.
package handler

import (
	"core-backend/internal/application"
)

type HandlerRegistry struct {
	ApplicationRegistry *application.ApplicationRegistry
	AuthHandler         *AuthHandler
	UserHandler         *UserHandler
	HealthHandler       *HealthHandler
	ProductHandler      *ProductHandler
	BrandHandler        *BrandHandler
	FileHandler         *S3Handler
	ContractHandler     *ContractHandler
}

func NewHandlerRegistry(applicationReg *application.ApplicationRegistry) *HandlerRegistry {
	return &HandlerRegistry{
		ApplicationRegistry: applicationReg,
		AuthHandler:         NewAuthHandler(applicationReg.AuthService),
		UserHandler:         NewUserHandler(applicationReg.UserService, applicationReg.InfrastructureRegistry.UnitOfWork),
		HealthHandler:       NewHealthHandler(applicationReg.InfrastructureRegistry),
		ProductHandler:      NewProductHandler(applicationReg.ProductService),
		BrandHandler:        NewBrandHandler(applicationReg.BrandService, applicationReg.InfrastructureRegistry.UnitOfWork),
		FileHandler:         NewS3Handler(applicationReg.FileService),
		ContractHandler:     NewContractHandler(applicationReg.ContractService, applicationReg.FileService, applicationReg.InfrastructureRegistry.UnitOfWork),
	}
}
