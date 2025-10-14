package middleware

import (
	"core-backend/config"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewCORSMiddleware() gin.HandlerFunc {
	corsConfig := config.GetAppConfig().CORS

	return cors.New(cors.Config{
		AllowOrigins:     corsConfig.AllowedOrigins,
		AllowMethods:     corsConfig.AllowedMethods,
		AllowHeaders:     corsConfig.AllowedHeaders,
		ExposeHeaders:    corsConfig.ExposedHeaders,
		AllowCredentials: corsConfig.AllowCredentials,
		AllowWebSockets:  true,
		MaxAge:           12 * time.Hour,
	})
}
