// Package persistence handles database connections and operations.
package persistence

import (
	"core-backend/config"
	"fmt"
	"log"
	"os"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB() *gorm.DB {
	zap.L().Info("Initializing database connection")

	dbCfg := config.GetAppConfig().Database
	zap.L().Debug("Database configuration loaded",
		zap.String("host", dbCfg.Host),
		zap.Int("port", dbCfg.Port),
		zap.String("dbname", dbCfg.DBName),
		zap.String("user", dbCfg.User),
		zap.String("sslmode", dbCfg.SSLMode))

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		dbCfg.Host,
		dbCfg.User,
		dbCfg.Password,
		dbCfg.DBName,
		dbCfg.Port,
		dbCfg.SSLMode,
	)

	// Configure GORM logger
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Set LogLevel to logger.Info to see all SQL queries
			IgnoreRecordNotFoundError: true,        // Don't log record not found errors
			Colorful:                  true,        // Enable color
		},
	)

	zap.L().Debug("Attempting to connect to PostgreSQL database")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil {
		zap.L().Error("Database connection failed",
			zap.String("host", dbCfg.Host),
			zap.Int("port", dbCfg.Port),
			zap.String("dbname", dbCfg.DBName),
			zap.Error(err))
		zap.L().Panic("Failed to connect to database", zap.Error(err))
	}

	zap.L().Info("Database connected successfully",
		zap.String("host", dbCfg.Host),
		zap.Int("port", dbCfg.Port),
		zap.String("dbname", dbCfg.DBName))

	// Verify database connection
	sqlDB, err := db.DB()
	if err != nil {
		zap.L().Error("Failed to get database instance", zap.Error(err))
		zap.L().Panic("Failed to get database instance", zap.Error(err))
	}

	if err := sqlDB.Ping(); err != nil {
		zap.L().Error("Database ping failed", zap.Error(err))
		zap.L().Panic("Database ping failed", zap.Error(err))
	}

	zap.L().Info("Database connection verified successfully")
	return db
}
