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
	FileHandler         *S3Handler
	PayOsHandler        *PayOsHandler
}

func NewHandlerRegistry(applicationReg *application.ApplicationRegistry) *HandlerRegistry {
	return &HandlerRegistry{
		ApplicationRegistry: applicationReg,
		AuthHandler:         NewAuthHandler(applicationReg.AuthService),
		UserHandler:         NewUserHandler(applicationReg.UserService),
		HealthHandler:       NewHealthHandler(applicationReg.InfrastructureRegistry),
		ProductHandler:      NewProductHandler(applicationReg.ProductService),
		FileHandler:         NewS3Handler(applicationReg.FileService),
		PayOsHandler:        NewPayOsHandler(applicationReg.InfrastructureRegistry.PayOsService),
	}
}
