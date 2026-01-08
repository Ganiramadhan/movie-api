package handlers

import (
	"context"
	"strconv"

	"movie-backend/internal/models"
	"movie-backend/internal/services"
	"movie-backend/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type MovieHandler struct {
	service services.MovieService
	logger  *logrus.Logger
}

func NewMovieHandler(service services.MovieService, logger *logrus.Logger) *MovieHandler {
	return &MovieHandler{
		service: service,
		logger:  logger,
	}
}

// GetAllMovies godoc
// @Summary Get all movies
// @Description Get list of all movies with pagination, search, sorting, and date range filter
// @Tags movies
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param search query string false "Search by title or overview"
// @Param sort_by query string false "Sort by field (id, title, release_date, vote_average, popularity, created_at, updated_at)" default(updated_at)
// @Param order query string false "Sort order (ASC/DESC)" default(DESC)
// @Param start_date query string false "Filter by start date (YYYY-MM-DD)"
// @Param end_date query string false "Filter by end date (YYYY-MM-DD)"
// @Success 200 {object} utils.StandardResponse "List of movies"
// @Failure 500 {object} utils.StandardResponse "Internal server error"
// @Router /movies [get]
func (h *MovieHandler) GetAllMovies(c *fiber.Ctx) error {
	ctx := c.Context()

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	search := c.Query("search", "")
	sortBy := c.Query("sort_by", "updated_at")
	order := c.Query("order", "DESC")
	startDate := c.Query("start_date", "")
	endDate := c.Query("end_date", "")

	movies, total, err := h.service.GetAllMovies(ctx, page, limit, search, sortBy, order, startDate, endDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get movies")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to retrieve movies")
	}

	meta := utils.CreatePaginationMeta(page, limit, total)
	return utils.SuccessWithMetaResponse(c, fiber.StatusOK, "Movies retrieved successfully", movies, meta)
}

// GetMovieByID godoc
// @Summary Get movie by ID
// @Description Get a single movie by its ID
// @Tags movies
// @Accept json
// @Produce json
// @Param id path int true "Movie ID"
// @Success 200 {object} utils.StandardResponse "Movie details"
// @Failure 400 {object} utils.StandardResponse "Invalid movie ID"
// @Failure 404 {object} utils.StandardResponse "Movie not found"
// @Router /movies/{id} [get]
func (h *MovieHandler) GetMovieByID(c *fiber.Ctx) error {
	ctx := c.Context()

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid movie ID")
	}

	movie, err := h.service.GetMovieByID(ctx, uint(id))
	if err != nil {
		h.logger.WithError(err).WithField("id", id).Error("Failed to get movie")
		return utils.ErrorResponse(c, fiber.StatusNotFound, "Movie not found")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Movie retrieved successfully", movie)
}

// CreateMovie godoc
// @Summary Create a new movie
// @Description Create a new movie entry
// @Tags movies
// @Accept json
// @Produce json
// @Param movie body MovieRequest true "Movie request object"
// @Success 201 {object} utils.StandardResponse "Movie created successfully"
// @Failure 400 {object} utils.StandardResponse "Invalid request body"
// @Failure 500 {object} utils.StandardResponse "Internal server error"
// @Router /movies [post]
func (h *MovieHandler) CreateMovie(c *fiber.Ctx) error {
	ctx := c.Context()

	var req MovieRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Convert request to movie model
	movie, err := h.convertRequestToMovie(ctx, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert request to movie")
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.service.CreateMovie(ctx, movie); err != nil {
		h.logger.WithError(err).Error("Failed to create movie")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, "Movie created successfully", movie)
}

// UpdateMovie godoc
// @Summary Update a movie
// @Description Update an existing movie
// @Tags movies
// @Accept json
// @Produce json
// @Param id path int true "Movie ID"
// @Param movie body MovieRequest true "Movie request object"
// @Success 200 {object} utils.StandardResponse "Movie updated successfully"
// @Failure 400 {object} utils.StandardResponse "Invalid request"
// @Failure 500 {object} utils.StandardResponse "Internal server error"
// @Router /movies/{id} [put]
func (h *MovieHandler) UpdateMovie(c *fiber.Ctx) error {
	ctx := c.Context()

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid movie ID")
	}

	var req MovieRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	movie, err := h.convertRequestToMovie(ctx, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert request to movie")
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.service.UpdateMovie(ctx, uint(id), movie); err != nil {
		h.logger.WithError(err).WithField("id", id).Error("Failed to update movie")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Movie updated successfully", movie)
}

// DeleteMovie godoc
// @Summary Delete a movie
// @Description Delete a movie by ID
// @Tags movies
// @Accept json
// @Produce json
// @Param id path int true "Movie ID"
// @Success 200 {object} utils.StandardResponse "Movie deleted successfully"
// @Failure 400 {object} utils.StandardResponse "Invalid movie ID"
// @Failure 500 {object} utils.StandardResponse "Internal server error"
// @Router /movies/{id} [delete]
func (h *MovieHandler) DeleteMovie(c *fiber.Ctx) error {
	ctx := c.Context()

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid movie ID")
	}

	if err := h.service.DeleteMovie(ctx, uint(id)); err != nil {
		h.logger.WithError(err).WithField("id", id).Error("Failed to delete movie")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Movie deleted successfully", nil)
}

// SyncMoviesFromTMDB godoc
// @Summary Sync movies from TMDB
// @Description Fetch and sync popular movies from TMDB API
// @Tags sync
// @Accept json
// @Produce json
// @Param pages query int false "Number of pages to sync (1-10)" default(1)
// @Success 200 {object} utils.StandardResponse "Sync completed successfully"
// @Failure 500 {object} utils.StandardResponse "Sync failed"
// @Router /sync/movies [post]
func (h *MovieHandler) SyncMoviesFromTMDB(c *fiber.Ctx) error {
	ctx := c.Context()

	pages, _ := strconv.Atoi(c.Query("pages", "1"))

	h.logger.WithField("pages", pages).Info("Starting TMDB sync")

	syncLog, err := h.service.SyncMoviesFromTMDB(ctx, pages)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sync movies from TMDB")
		return utils.ErrorWithDataResponse(c, fiber.StatusInternalServerError, "Failed to sync movies", syncLog)
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Movies synced successfully", syncLog)
}

// GetDashboardStats godoc
// @Summary Get dashboard statistics
// @Description Get comprehensive dashboard analytics
// @Tags dashboard
// @Accept json
// @Produce json
// @Success 200 {object} utils.StandardResponse "Dashboard statistics"
// @Failure 500 {object} utils.StandardResponse "Failed to retrieve statistics"
// @Router /dashboard/stats [get]
func (h *MovieHandler) GetDashboardStats(c *fiber.Ctx) error {
	ctx := c.Context()

	stats, err := h.service.GetDashboardStats(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get dashboard stats")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to retrieve dashboard statistics")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Dashboard statistics retrieved successfully", stats)
}

// GetLastSyncLog godoc
// @Summary Get last sync log
// @Description Get the most recent sync operation log
// @Tags sync
// @Accept json
// @Produce json
// @Success 200 {object} utils.StandardResponse "Last sync log"
// @Failure 500 {object} utils.StandardResponse "Failed to retrieve sync log"
// @Router /sync/last-log [get]
func (h *MovieHandler) GetLastSyncLog(c *fiber.Ctx) error {
	ctx := c.Context()

	syncLog, err := h.service.GetLastSyncLog(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get last sync log")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to retrieve last sync log")
	}

	if syncLog == nil {
		return utils.SuccessResponse(c, fiber.StatusOK, "No sync log found", nil)
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Last sync log retrieved successfully", syncLog)
}

// GetChartData godoc
// @Summary Get chart data for visualization
// @Description Get combined pie chart (by language) and column chart (by year) data
// @Tags charts
// @Accept json
// @Produce json
// @Param start_date query string false "Filter by start date (YYYY-MM-DD)"
// @Param end_date query string false "Filter by end date (YYYY-MM-DD)"
// @Success 200 {object} utils.StandardResponse "Chart data"
// @Failure 500 {object} utils.StandardResponse "Failed to retrieve chart data"
// @Router /charts [get]
func (h *MovieHandler) GetChartData(c *fiber.Ctx) error {
	ctx := c.Context()

	startDate := c.Query("start_date", "")
	endDate := c.Query("end_date", "")

	chartData, err := h.service.GetChartData(ctx, startDate, endDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get chart data")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to retrieve chart data")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Chart data retrieved successfully", chartData)
}

// GetPieChartData godoc
// @Summary Get pie chart data by language
// @Description Get movie distribution by original language for pie chart visualization
// @Tags charts
// @Accept json
// @Produce json
// @Success 200 {object} utils.StandardResponse "Pie chart data"
// @Failure 500 {object} utils.StandardResponse "Failed to retrieve pie chart data"
// @Router /charts/pie [get]
func (h *MovieHandler) GetPieChartData(c *fiber.Ctx) error {
	ctx := c.Context()

	data, err := h.service.GetMoviesByLanguage(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get pie chart data")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to retrieve pie chart data")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Pie chart data retrieved successfully", data)
}

// GetColumnChartData godoc
// @Summary Get column chart data by year
// @Description Get movie distribution by release year for column chart visualization
// @Tags charts
// @Accept json
// @Produce json
// @Param start_date query string false "Filter by start date (YYYY-MM-DD)"
// @Param end_date query string false "Filter by end date (YYYY-MM-DD)"
// @Success 200 {object} utils.StandardResponse "Column chart data"
// @Failure 500 {object} utils.StandardResponse "Failed to retrieve column chart data"
// @Router /charts/column [get]
func (h *MovieHandler) GetColumnChartData(c *fiber.Ctx) error {
	ctx := c.Context()

	startDate := c.Query("start_date", "")
	endDate := c.Query("end_date", "")

	data, err := h.service.GetMoviesByYear(ctx, startDate, endDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get column chart data")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to retrieve column chart data")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Column chart data retrieved successfully", data)
}

// GetMonthlyChartData godoc
// @Summary Get monthly chart data for a specific year
// @Description Get movie distribution by month for a specific year
// @Tags charts
// @Accept json
// @Produce json
// @Param year path int true "Year (e.g., 2024)"
// @Success 200 {object} utils.StandardResponse "Monthly chart data"
// @Failure 400 {object} utils.StandardResponse "Invalid year"
// @Failure 500 {object} utils.StandardResponse "Failed to retrieve monthly chart data"
// @Router /charts/monthly/{year} [get]
func (h *MovieHandler) GetMonthlyChartData(c *fiber.Ctx) error {
	ctx := c.Context()

	year, err := strconv.Atoi(c.Params("year"))
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid year format")
	}

	data, err := h.service.GetMoviesByMonth(ctx, year)
	if err != nil {
		h.logger.WithError(err).WithField("year", year).Error("Failed to get monthly chart data")
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to retrieve monthly chart data")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Monthly chart data retrieved successfully", data)
}

func (h *MovieHandler) convertRequestToMovie(ctx context.Context, req *MovieRequest) (*models.Movie, error) {
	langSvc, ok := h.service.(interface {
		GetLanguageByCode(context.Context, string) (*models.Language, error)
		CreateLanguage(context.Context, string, string) (*models.Language, error)
	})

	var languageID *uint
	if ok && req.OriginalLanguage != "" {
		lang, err := langSvc.GetLanguageByCode(ctx, req.OriginalLanguage)
		if err == nil && lang != nil {
			languageID = &lang.ID
		} else {
			langName := getLanguageName(req.OriginalLanguage)
			lang, err = langSvc.CreateLanguage(ctx, req.OriginalLanguage, langName)
			if err == nil && lang != nil {
				languageID = &lang.ID
			}
		}
	}

	movie := &models.Movie{
		TMDBID:        req.TMDBID,
		Title:         req.Title,
		OriginalTitle: req.OriginalTitle,
		Overview:      req.Overview,
		ReleaseDate:   req.ReleaseDate,
		PosterPath:    req.PosterPath,
		BackdropPath:  req.BackdropPath,
		VoteAverage:   req.VoteAverage,
		VoteCount:     req.VoteCount,
		Popularity:    req.Popularity,
		Adult:         req.Adult,
		LanguageID:    languageID,
	}

	return movie, nil
}

func getLanguageName(langCode string) string {
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
