package repository

import (
	"context"
	"errors"
	"time"

	"movie-backend/internal/database"
	"movie-backend/internal/models"

	"gorm.io/gorm"
)

type MovieRepository interface {
	// CRUD operations
	Create(ctx context.Context, movie *models.Movie) error
	Update(ctx context.Context, movie *models.Movie) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*models.Movie, error)
	FindByTMDBID(ctx context.Context, tmdbID int) (*models.Movie, error)
	FindAll(ctx context.Context, page, limit int, search, sortBy, order, startDate, endDate string) ([]models.Movie, int64, error)

	// Dashboard operations
	GetDashboardStats(ctx context.Context) (*models.DashboardStats, error)

	// Sync log operations
	CreateSyncLog(ctx context.Context, log *models.SyncLog) error
	GetLastSyncLog(ctx context.Context) (*models.SyncLog, error)

	// Chart data operations
	GetMoviesByLanguage(ctx context.Context) ([]models.PieChartData, error)
	GetMoviesByYear(ctx context.Context, startDate, endDate string) ([]models.ColumnChartData, error)
	GetMoviesByMonth(ctx context.Context, year int) ([]models.ColumnChartData, error)
}

type movieRepository struct {
	db      *database.Database
	timeout time.Duration
}

func NewMovieRepository(db *database.Database) MovieRepository {
	return &movieRepository{
		db:      db,
		timeout: db.GetQueryTimeout(),
	}
}

func (r *movieRepository) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, r.timeout)
}

func (r *movieRepository) Create(ctx context.Context, movie *models.Movie) error {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	return r.db.WithContext(ctx).Create(movie).Error
}

func (r *movieRepository) Update(ctx context.Context, movie *models.Movie) error {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	return r.db.WithContext(ctx).Save(movie).Error
}

func (r *movieRepository) Delete(ctx context.Context, id uint) error {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	return r.db.WithContext(ctx).Delete(&models.Movie{}, id).Error
}

func (r *movieRepository) FindByID(ctx context.Context, id uint) (*models.Movie, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var movie models.Movie
	err := r.db.WithContext(ctx).Preload("Language").Preload("Genres").First(&movie, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("movie not found")
		}
		return nil, err
	}
	return &movie, nil
}

func (r *movieRepository) FindByTMDBID(ctx context.Context, tmdbID int) (*models.Movie, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var movie models.Movie
	err := r.db.WithContext(ctx).Where("tmdb_id = ?", tmdbID).First(&movie).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &movie, nil
}

func (r *movieRepository) FindAll(ctx context.Context, page, limit int, search, sortBy, order, startDate, endDate string) ([]models.Movie, int64, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var movies []models.Movie
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Movie{})

	// Apply search filter
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("title ILIKE ? OR overview ILIKE ? OR original_title ILIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	// Apply date range filter
	if startDate != "" {
		query = query.Where("release_date >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("release_date <= ?", endDate)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting with validation
	validSortFields := map[string]bool{
		"id": true, "title": true, "release_date": true, "vote_average": true,
		"popularity": true, "created_at": true, "updated_at": true,
	}
	if !validSortFields[sortBy] {
		sortBy = "updated_at"
	}
	if order != "ASC" && order != "asc" {
		order = "DESC"
	}
	query = query.Order(sortBy + " " + order)

	// Apply pagination
	offset := (page - 1) * limit
	if err := query.Preload("Language").Preload("Genres").Offset(offset).Limit(limit).Find(&movies).Error; err != nil {
		return nil, 0, err
	}

	return movies, total, nil
}

func (r *movieRepository) GetDashboardStats(ctx context.Context) (*models.DashboardStats, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var stats models.DashboardStats
	db := r.db.WithContext(ctx)

	// Total movies
	if err := db.Model(&models.Movie{}).Count(&stats.TotalMovies).Error; err != nil {
		return nil, err
	}

	if stats.TotalMovies > 0 {
		type AggResult struct {
			AvgRating  float64
			TotalVotes int64
		}
		var result AggResult
		if err := db.Model(&models.Movie{}).
			Select("COALESCE(AVG(vote_average), 0) as avg_rating, COALESCE(SUM(vote_count), 0) as total_votes").
			Scan(&result).Error; err != nil {
			return nil, err
		}
		stats.AverageRating = result.AvgRating
		stats.TotalVotes = result.TotalVotes
	}

	// Last sync time
	var lastSync models.SyncLog
	if err := db.Model(&models.SyncLog{}).Order("synced_at DESC").First(&lastSync).Error; err == nil {
		stats.LastSyncTime = &lastSync.SyncedAt
	}

	// Top rated movies (limit 10)
	if err := db.Model(&models.Movie{}).
		Preload("Language").Preload("Genres").
		Where("vote_count > ?", 100). // Only movies with significant votes
		Order("vote_average DESC, vote_count DESC").
		Limit(10).
		Find(&stats.TopRatedMovies).Error; err != nil {
		return nil, err
	}

	// Most popular movies (limit 10)
	if err := db.Model(&models.Movie{}).
		Preload("Language").Preload("Genres").
		Order("popularity DESC").
		Limit(10).
		Find(&stats.MostPopular).Error; err != nil {
		return nil, err
	}

	// Recently added movies (limit 10)
	if err := db.Model(&models.Movie{}).
		Preload("Language").Preload("Genres").
		Order("created_at DESC").
		Limit(10).
		Find(&stats.RecentlyAdded).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *movieRepository) CreateSyncLog(ctx context.Context, log *models.SyncLog) error {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	return r.db.WithContext(ctx).Create(log).Error
}

func (r *movieRepository) GetLastSyncLog(ctx context.Context) (*models.SyncLog, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var log models.SyncLog
	err := r.db.WithContext(ctx).Order("synced_at DESC").First(&log).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &log, nil
}

func (r *movieRepository) GetMoviesByLanguage(ctx context.Context) ([]models.PieChartData, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var results []models.PieChartData

	err := r.db.WithContext(ctx).Model(&models.Movie{}).
		Select("COALESCE(languages.name, 'Unknown') as label, COALESCE(languages.code, 'unknown') as code, COUNT(movies.id) as value").
		Joins("LEFT JOIN languages ON movies.language_id = languages.id").
		Group("languages.name, languages.code").
		Order("value DESC").
		Limit(10).
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *movieRepository) GetMoviesByYear(ctx context.Context, startDate, endDate string) ([]models.ColumnChartData, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var results []models.ColumnChartData

	query := r.db.WithContext(ctx).Model(&models.Movie{}).
		Select("SUBSTRING(release_date, 1, 4) as label, COUNT(*) as value").
		Where("release_date != '' AND release_date IS NOT NULL AND LENGTH(release_date) >= 4")

	if startDate != "" {
		query = query.Where("release_date >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("release_date <= ?", endDate)
	}

	err := query.Group("SUBSTRING(release_date, 1, 4)").
		Order("label DESC").
		Limit(10).
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *movieRepository) GetMoviesByMonth(ctx context.Context, year int) ([]models.ColumnChartData, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var results []models.ColumnChartData

	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

	type MonthCount struct {
		Month int64
		Count int64
	}

	var monthCounts []MonthCount
	yearStr := string(rune('0'+year/1000)) + string(rune('0'+(year/100)%10)) + string(rune('0'+(year/10)%10)) + string(rune('0'+year%10))

	err := r.db.WithContext(ctx).Model(&models.Movie{}).
		Select("CAST(SUBSTRING(release_date, 6, 2) AS INTEGER) as month, COUNT(*) as count").
		Where("release_date LIKE ?", yearStr+"%").
		Where("LENGTH(release_date) >= 7").
		Group("SUBSTRING(release_date, 6, 2)").
		Order("month").
		Find(&monthCounts).Error

	if err != nil {
		return nil, err
	}

	monthMap := make(map[int64]int64)
	for _, mc := range monthCounts {
		monthMap[mc.Month] = mc.Count
	}

	for i, month := range months {
		results = append(results, models.ColumnChartData{
			Label: month,
			Value: monthMap[int64(i+1)],
		})
	}

	return results, nil
}
