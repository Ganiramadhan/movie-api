package repository

import (
	"context"
	"errors"
	"time"

	"movie-backend/internal/database"
	"movie-backend/internal/models"

	"gorm.io/gorm"
)

type GenreRepository interface {
	Create(ctx context.Context, genre *models.Genre) error
	FindByTMDBID(ctx context.Context, tmdbID int) (*models.Genre, error)
	FindOrCreate(ctx context.Context, tmdbID int, name string) (*models.Genre, error)
	FindAll(ctx context.Context) ([]models.Genre, error)
}

type genreRepository struct {
	db      *database.Database
	timeout time.Duration
}

func NewGenreRepository(db *database.Database) GenreRepository {
	return &genreRepository{
		db:      db,
		timeout: db.GetQueryTimeout(),
	}
}

func (r *genreRepository) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, r.timeout)
}

func (r *genreRepository) Create(ctx context.Context, genre *models.Genre) error {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	return r.db.WithContext(ctx).Create(genre).Error
}

func (r *genreRepository) FindByTMDBID(ctx context.Context, tmdbID int) (*models.Genre, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var genre models.Genre
	err := r.db.WithContext(ctx).Where("tmdb_id = ?", tmdbID).First(&genre).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &genre, nil
}

func (r *genreRepository) FindOrCreate(ctx context.Context, tmdbID int, name string) (*models.Genre, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var genre models.Genre
	err := r.db.WithContext(ctx).Where("tmdb_id = ?", tmdbID).FirstOrCreate(&genre, models.Genre{
		TMDBID: tmdbID,
		Name:   name,
	}).Error
	if err != nil {
		return nil, err
	}
	return &genre, nil
}

func (r *genreRepository) FindAll(ctx context.Context) ([]models.Genre, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var genres []models.Genre
	err := r.db.WithContext(ctx).Find(&genres).Error
	return genres, err
}
