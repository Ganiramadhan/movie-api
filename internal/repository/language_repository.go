package repository

import (
	"context"
	"errors"
	"time"

	"movie-backend/internal/database"
	"movie-backend/internal/models"

	"gorm.io/gorm"
)

type LanguageRepository interface {
	Create(ctx context.Context, language *models.Language) error
	FindByCode(ctx context.Context, code string) (*models.Language, error)
	FindOrCreate(ctx context.Context, code, name string) (*models.Language, error)
	FindAll(ctx context.Context) ([]models.Language, error)
}

type languageRepository struct {
	db      *database.Database
	timeout time.Duration
}

func NewLanguageRepository(db *database.Database) LanguageRepository {
	return &languageRepository{
		db:      db,
		timeout: db.GetQueryTimeout(),
	}
}

func (r *languageRepository) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, r.timeout)
}

func (r *languageRepository) Create(ctx context.Context, language *models.Language) error {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	return r.db.WithContext(ctx).Create(language).Error
}

func (r *languageRepository) FindByCode(ctx context.Context, code string) (*models.Language, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var language models.Language
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&language).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &language, nil
}

func (r *languageRepository) FindOrCreate(ctx context.Context, code, name string) (*models.Language, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var language models.Language
	err := r.db.WithContext(ctx).Where("code = ?", code).FirstOrCreate(&language, models.Language{
		Code: code,
		Name: name,
	}).Error
	if err != nil {
		return nil, err
	}
	return &language, nil
}

func (r *languageRepository) FindAll(ctx context.Context) ([]models.Language, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var languages []models.Language
	err := r.db.WithContext(ctx).Find(&languages).Error
	return languages, err
}
