// Package persistence handles database connections and operations.
package persistence

import (
	"fmt"
	"core-backend/config"
	"core-backend/internal/domain/model"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	dbCfg := config.GetAppConfig().Database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		dbCfg.Host,
		dbCfg.User,
		dbCfg.Password,
		dbCfg.DBName,
		dbCfg.Port,
		dbCfg.SSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		zap.L().Panic("Failed to connect to database", zap.Error(err))
	}

	zap.L().Info("Database connected", zap.String("host", dbCfg.Host), zap.Int("port", dbCfg.Port), zap.String("dbname", dbCfg.DBName))

	// Auto migrate all models
	err = db.AutoMigrate(
		&model.User{},
		&model.LoggedSession{},
		&model.Brand{},
		&model.Contract{},
		&model.Campaign{},
	)
	if err != nil {
		zap.L().Panic("Failed to migrate database", zap.Error(err))
	}

	return db
}
