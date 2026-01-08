package database

import (
	"context"
	"fmt"
	"time"

	"movie-backend/internal/config"
	"movie-backend/internal/models"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	*gorm.DB
	config config.DatabaseConfig
}

func Connect(cfg config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC connect_timeout=10",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true, // Enable prepared statement cache
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		logrus.WithError(err).Error("Failed to connect to database")
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logrus.WithError(err).Error("Failed to get underlying sql.DB")
		return nil, fmt.Errorf("failed to get underlying sql.DB: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		logrus.WithError(err).Error("Failed to ping database")
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	logrus.Info("Database connection established successfully")

	database := &Database{
		DB:     db,
		config: cfg,
	}

	// Auto migrate models
	if err := autoMigrate(db); err != nil {
		logrus.WithError(err).Error("Failed to run auto migration")
		return nil, fmt.Errorf("failed to run auto migration: %v", err)
	}

	return database, nil
}

func (d *Database) WithContext(ctx context.Context) *gorm.DB {
	return d.DB.WithContext(ctx)
}

func (d *Database) WithTimeout() (*gorm.DB, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), d.config.QueryTimeout)
	return d.DB.WithContext(ctx), cancel
}

func (d *Database) GetQueryTimeout() time.Duration {
	return d.config.QueryTimeout
}

func (d *Database) HealthCheck() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return sqlDB.PingContext(ctx)
}

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func autoMigrate(db *gorm.DB) error {
	logrus.Info("Running auto migration...")

	err := db.AutoMigrate(
		&models.Movie{},
		&models.SyncLog{},
		&models.Genre{},
		&models.Language{},
		&models.MovieGenre{},
	)

	if err != nil {
		return err
	}

	logrus.Info("Auto migration completed successfully")
	return nil
}
