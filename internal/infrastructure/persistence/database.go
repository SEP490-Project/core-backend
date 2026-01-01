// Package persistence handles database connections and operations.
package persistence

import (
	"core-backend/config"
	"core-backend/pkg/logging"
	"database/sql"
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
	var gormLogger logger.Interface
	if config.GetAppConfig().IsDevelopment() {
		gormLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  logger.Info, // Set LogLevel to logger.Info to see all SQL queries
				IgnoreRecordNotFoundError: true,        // Don't log record not found errors
				Colorful:                  true,        // Enable color
			},
		)
	} else {
		loggerImpl := logging.NewZapGormLogger(zap.L())
		loggerImpl.SlowThreshold = time.Second
		loggerImpl.IgnoreRecordNotFoundError = false
		gormLogger = &loggerImpl
	}

	gormConfig := &gorm.Config{
		Logger:      gormLogger,
		PrepareStmt: false,
	}

	var db *gorm.DB
	var err error
	zap.L().Debug("Attempting to connect to PostgreSQL database")
	for i := range 5 {
		db, err = gorm.Open(postgres.Open(dsn), gormConfig)
		if err == nil {
			var sqlDB *sql.DB
			sqlDB, err = db.DB()
			if err != nil {
				zap.L().Error("Failed to get database instance", zap.Error(err))
			}

			// Set connection pool settings
			sqlDB.SetMaxOpenConns(dbCfg.Pool.MaxOpenConns)
			sqlDB.SetMaxIdleConns(dbCfg.Pool.MaxIdleConns)
			sqlDB.SetConnMaxLifetime(time.Second * time.Duration(dbCfg.Pool.ConnMaxLifetime))
			sqlDB.SetConnMaxIdleTime(time.Second * time.Duration(dbCfg.Pool.ConnMaxIdleTime))

			// Verify the connection via ping
			err = sqlDB.Ping()
			if err == nil {
				zap.L().Info("Database connection verified successfully",
					zap.String("host", dbCfg.Host),
					zap.Int("port", dbCfg.Port),
					zap.String("dbname", dbCfg.DBName))
				return db
			}
			zap.L().Error("Database ping failed", zap.Error(err))

		}

		zap.S().Errorf("Attempt %d: Failed to connect to database, retrying after %d seconds", i+1, i+1)
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	zap.L().Error("All attempts to connect to the database have failed",
		zap.String("host", dbCfg.Host),
		zap.Int("port", dbCfg.Port),
		zap.String("dbname", dbCfg.DBName),
		zap.Error(err))
	zap.L().Panic("Failed to connect to database after multiple attempts", zap.Error(err))
	return nil
}
