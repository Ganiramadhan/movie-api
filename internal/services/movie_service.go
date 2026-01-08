package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"movie-backend/internal/config"
	"movie-backend/internal/models"
	"movie-backend/internal/repository"

	"github.com/sirupsen/logrus"
)

type MovieService interface {
	// CRUD operations
	CreateMovie(ctx context.Context, movie *models.Movie) error
	UpdateMovie(ctx context.Context, id uint, movie *models.Movie) error
	DeleteMovie(ctx context.Context, id uint) error
	GetMovieByID(ctx context.Context, id uint) (*models.Movie, error)
	GetAllMovies(ctx context.Context, page, limit int, search, sortBy, order, startDate, endDate string) ([]models.Movie, int64, error)

	// Sync operations
	SyncMoviesFromTMDB(ctx context.Context, pages int) (*models.SyncLog, error)
	GetLastSyncLog(ctx context.Context) (*models.SyncLog, error)

	// Dashboard operations
	GetDashboardStats(ctx context.Context) (*models.DashboardStats, error)

	// Chart data operations
	GetChartData(ctx context.Context, startDate, endDate string) (*models.ChartDataResponse, error)
	GetMoviesByLanguage(ctx context.Context) ([]models.PieChartData, error)
	GetMoviesByYear(ctx context.Context, startDate, endDate string) ([]models.ColumnChartData, error)
	GetMoviesByMonth(ctx context.Context, year int) ([]models.ColumnChartData, error)

	// Language operations
	GetLanguageByCode(ctx context.Context, code string) (*models.Language, error)
	CreateLanguage(ctx context.Context, code, name string) (*models.Language, error)
}

type movieService struct {
	repo         repository.MovieRepository
	genreRepo    repository.GenreRepository
	langRepo     repository.LanguageRepository
	config       *config.Config
	logger       *logrus.Logger
	httpClient   *http.Client
	minioService *MinIOService
}

func NewMovieService(repo repository.MovieRepository, genreRepo repository.GenreRepository, langRepo repository.LanguageRepository, cfg *config.Config, logger *logrus.Logger) MovieService {
	return &movieService{
		repo:      repo,
		genreRepo: genreRepo,
		langRepo:  langRepo,
		config:    cfg,
		logger:    logger,
		httpClient: &http.Client{
			Timeout: cfg.TMDB.HTTPTimeout,
		},
	}
}

func (s *movieService) SetMinIOService(minioSvc *MinIOService) {
	s.minioService = minioSvc
}

func (s *movieService) CreateMovie(ctx context.Context, movie *models.Movie) error {
	if movie.Title == "" {
		return fmt.Errorf("movie title is required")
	}

	// Check if movie with same TMDB ID already exists
	if movie.TMDBID > 0 {
		existing, err := s.repo.FindByTMDBID(ctx, movie.TMDBID)
		if err != nil {
			return fmt.Errorf("failed to check existing movie: %w", err)
		}
		if existing != nil {
			return fmt.Errorf("movie with TMDB ID %d already exists", movie.TMDBID)
		}
	}

	return s.repo.Create(ctx, movie)
}

func (s *movieService) UpdateMovie(ctx context.Context, id uint, movie *models.Movie) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("movie with ID %d not found", id)
	}

	// If image is being updated and old image is MinIO URL, delete it
	if s.minioService != nil {
		// Delete old poster if being replaced
		if movie.PosterPath != "" && movie.PosterPath != existing.PosterPath {
			if strings.Contains(existing.PosterPath, "http") && strings.Contains(existing.PosterPath, s.config.MinIO.BucketName) {
				// Extract filename from URL
				parts := strings.Split(existing.PosterPath, "/")
				if len(parts) > 0 {
					filename := parts[len(parts)-1]
					// Remove query params if any (presigned URL)
					if idx := strings.Index(filename, "?"); idx != -1 {
						filename = filename[:idx]
					}
					if err := s.minioService.DeleteFile(filename); err != nil {
						s.logger.WithError(err).Warn("Failed to delete old poster from MinIO")
					}
				}
			}
		}

		// Delete old backdrop if being replaced
		if movie.BackdropPath != "" && movie.BackdropPath != existing.BackdropPath {
			if strings.Contains(existing.BackdropPath, "http") && strings.Contains(existing.BackdropPath, s.config.MinIO.BucketName) {
				// Extract filename from URL
				parts := strings.Split(existing.BackdropPath, "/")
				if len(parts) > 0 {
					filename := parts[len(parts)-1]
					// Remove query params if any (presigned URL)
					if idx := strings.Index(filename, "?"); idx != -1 {
						filename = filename[:idx]
					}
					if err := s.minioService.DeleteFile(filename); err != nil {
						s.logger.WithError(err).Warn("Failed to delete old backdrop from MinIO")
					}
				}
			}
		}
	}

	movie.ID = id
	movie.CreatedAt = existing.CreatedAt
	movie.TMDBID = existing.TMDBID // Don't allow changing TMDB ID

	return s.repo.Update(ctx, movie)
}

func (s *movieService) DeleteMovie(ctx context.Context, id uint) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("movie with ID %d not found", id)
	}

	// Delete images from MinIO if they are MinIO URLs
	if s.minioService != nil {
		// Delete poster
		if existing.PosterPath != "" {
			if strings.Contains(existing.PosterPath, "http") && strings.Contains(existing.PosterPath, s.config.MinIO.BucketName) {
				// Extract filename from URL
				parts := strings.Split(existing.PosterPath, "/")
				if len(parts) > 0 {
					filename := parts[len(parts)-1]
					// Remove query params if any (presigned URL)
					if idx := strings.Index(filename, "?"); idx != -1 {
						filename = filename[:idx]
					}
					if err := s.minioService.DeleteFile(filename); err != nil {
						s.logger.WithError(err).Warn("Failed to delete poster from MinIO")
					}
				}
			}
		}

		// Delete backdrop
		if existing.BackdropPath != "" {
			if strings.Contains(existing.BackdropPath, "http") && strings.Contains(existing.BackdropPath, s.config.MinIO.BucketName) {
				// Extract filename from URL
				parts := strings.Split(existing.BackdropPath, "/")
				if len(parts) > 0 {
					filename := parts[len(parts)-1]
					// Remove query params if any (presigned URL)
					if idx := strings.Index(filename, "?"); idx != -1 {
						filename = filename[:idx]
					}
					if err := s.minioService.DeleteFile(filename); err != nil {
						s.logger.WithError(err).Warn("Failed to delete backdrop from MinIO")
					}
				}
			}
		}
	}

	return s.repo.Delete(ctx, id)
}

func (s *movieService) GetMovieByID(ctx context.Context, id uint) (*models.Movie, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *movieService) GetAllMovies(ctx context.Context, page, limit int, search, sortBy, order, startDate, endDate string) ([]models.Movie, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return s.repo.FindAll(ctx, page, limit, search, sortBy, order, startDate, endDate)
}

func (s *movieService) SyncMoviesFromTMDB(ctx context.Context, pages int) (*models.SyncLog, error) {
	syncLog := &models.SyncLog{
		SyncType: "manual",
		Status:   "failed",
		SyncedAt: time.Now().UTC(),
	}

	// Validate pages
	if pages < 1 {
		pages = 1
	}
	if pages > 10 {
		pages = 10 // Limit to prevent too many API calls
	}

	var moviesAdded, moviesUpdated int

	for page := 1; page <= pages; page++ {
		s.logger.WithField("page", page).Info("Fetching TMDB popular movies")

		movies, err := s.fetchPopularMoviesFromTMDB(ctx, page)
		if err != nil {
			syncLog.ErrorMessage = fmt.Sprintf("failed to fetch page %d: %s", page, err.Error())
			_ = s.repo.CreateSyncLog(ctx, syncLog)
			return syncLog, err
		}

		for _, tmdbMovie := range movies {
			// Get or create language
			langCode := tmdbMovie.OriginalLanguage
			langName := s.getLanguageName(langCode)
			language, err := s.langRepo.FindOrCreate(ctx, langCode, langName)
			if err != nil {
				s.logger.WithError(err).WithField("lang_code", langCode).Error("Error creating language")
				continue
			}

			movie := &models.Movie{
				TMDBID:        tmdbMovie.ID,
				Title:         tmdbMovie.Title,
				OriginalTitle: tmdbMovie.OriginalTitle,
				Overview:      tmdbMovie.Overview,
				ReleaseDate:   tmdbMovie.ReleaseDate,
				PosterPath:    tmdbMovie.PosterPath,
				BackdropPath:  tmdbMovie.BackdropPath,
				VoteAverage:   tmdbMovie.VoteAverage,
				VoteCount:     tmdbMovie.VoteCount,
				Popularity:    tmdbMovie.Popularity,
				Adult:         tmdbMovie.Adult,
				LanguageID:    &language.ID,
			}

			// Get or create genres
			var genres []models.Genre
			for _, genreID := range tmdbMovie.GenreIDs {
				genreName := s.getGenreName(genreID)
				genre, err := s.genreRepo.FindOrCreate(ctx, genreID, genreName)
				if err != nil {
					s.logger.WithError(err).WithField("genre_id", genreID).Error("Error creating genre")
					continue
				}
				genres = append(genres, *genre)
			}
			movie.Genres = genres

			// Check if movie already exists
			existing, err := s.repo.FindByTMDBID(ctx, movie.TMDBID)
			if err != nil {
				s.logger.WithError(err).WithField("tmdb_id", movie.TMDBID).Error("Error checking existing movie")
				continue
			}

			if existing == nil {
				// Create new movie
				if err := s.repo.Create(ctx, movie); err != nil {
					s.logger.WithError(err).WithField("title", movie.Title).Error("Error creating movie")
					continue
				}
				moviesAdded++
			} else {
				// Update existing movie
				movie.ID = existing.ID
				movie.CreatedAt = existing.CreatedAt
				if err := s.repo.Update(ctx, movie); err != nil {
					s.logger.WithError(err).WithField("title", movie.Title).Error("Error updating movie")
					continue
				}
				moviesUpdated++
			}
		}
	}

	syncLog.Status = "success"
	syncLog.MoviesAdded = moviesAdded
	syncLog.MoviesUpdated = moviesUpdated
	_ = s.repo.CreateSyncLog(ctx, syncLog)

	s.logger.WithFields(logrus.Fields{
		"movies_added":   moviesAdded,
		"movies_updated": moviesUpdated,
	}).Info("Sync completed")

	return syncLog, nil
}

func (s *movieService) fetchPopularMoviesFromTMDB(ctx context.Context, page int) ([]models.TMDBMovieResponse, error) {
	url := fmt.Sprintf("%s/movie/popular?api_key=%s&page=%d&language=en-US",
		s.config.TMDB.BaseURL,
		s.config.TMDB.APIKey,
		page,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from TMDB: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDB API returned status %d: %s", resp.StatusCode, string(body))
	}

	var tmdbResponse models.TMDBPopularMoviesResponse
	if err := json.NewDecoder(resp.Body).Decode(&tmdbResponse); err != nil {
		return nil, fmt.Errorf("failed to decode TMDB response: %w", err)
	}

	return tmdbResponse.Results, nil
}

// getGenreName returns the genre name for a given TMDB genre ID
func (s *movieService) getGenreName(genreID int) string {
	genreMap := map[int]string{
		28: "Action", 12: "Adventure", 16: "Animation", 35: "Comedy", 80: "Crime",
		99: "Documentary", 18: "Drama", 10751: "Family", 14: "Fantasy", 36: "History",
		27: "Horror", 10402: "Music", 9648: "Mystery", 10749: "Romance", 878: "Science Fiction",
		10770: "TV Movie", 53: "Thriller", 10752: "War", 37: "Western",
	}
	if name, ok := genreMap[genreID]; ok {
		return name
	}
	return fmt.Sprintf("Genre %d", genreID)
}

// getLanguageName returns the language name for a given language code
func (s *movieService) getLanguageName(langCode string) string {
	langMap := map[string]string{
		"en": "English", "ja": "Japanese", "ko": "Korean", "zh": "Chinese",
		"es": "Spanish", "fr": "French", "de": "German", "it": "Italian",
		"pt": "Portuguese", "ru": "Russian", "hi": "Hindi", "th": "Thai",
		"id": "Indonesian", "tr": "Turkish", "ar": "Arabic", "pl": "Polish",
		"nl": "Dutch", "sv": "Swedish", "no": "Norwegian", "da": "Danish",
		"fi": "Finnish", "cs": "Czech", "hu": "Hungarian", "ro": "Romanian",
	}
	if name, ok := langMap[langCode]; ok {
		return name
	}
	return langCode
}

func (s *movieService) GetDashboardStats(ctx context.Context) (*models.DashboardStats, error) {
	return s.repo.GetDashboardStats(ctx)
}

func (s *movieService) GetLastSyncLog(ctx context.Context) (*models.SyncLog, error) {
	return s.repo.GetLastSyncLog(ctx)
}

// GetChartData returns combined chart data for visualization
func (s *movieService) GetChartData(ctx context.Context, startDate, endDate string) (*models.ChartDataResponse, error) {
	pieData, err := s.repo.GetMoviesByLanguage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pie chart data: %w", err)
	}

	columnData, err := s.repo.GetMoviesByYear(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get column chart data: %w", err)
	}

	return &models.ChartDataResponse{
		PieChart:    pieData,
		ColumnChart: columnData,
	}, nil
}

// GetMoviesByLanguage returns movie distribution by language
func (s *movieService) GetMoviesByLanguage(ctx context.Context) ([]models.PieChartData, error) {
	return s.repo.GetMoviesByLanguage(ctx)
}

// GetMoviesByYear returns movie distribution by year
func (s *movieService) GetMoviesByYear(ctx context.Context, startDate, endDate string) ([]models.ColumnChartData, error) {
	return s.repo.GetMoviesByYear(ctx, startDate, endDate)
}

// GetMoviesByMonth returns movie distribution by month for a specific year
func (s *movieService) GetMoviesByMonth(ctx context.Context, year int) ([]models.ColumnChartData, error) {
	if year < 1900 || year > 2100 {
		return nil, fmt.Errorf("invalid year: %d", year)
	}
	return s.repo.GetMoviesByMonth(ctx, year)
}

// GetLanguageByCode returns language by code
func (s *movieService) GetLanguageByCode(ctx context.Context, code string) (*models.Language, error) {
	return s.langRepo.FindByCode(ctx, code)
}

// CreateLanguage creates a new language
func (s *movieService) CreateLanguage(ctx context.Context, code, name string) (*models.Language, error) {
	return s.langRepo.FindOrCreate(ctx, code, name)
}
