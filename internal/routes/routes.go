package routes

import (
	"movie-backend/internal/handlers"

	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App, movieHandler *handlers.MovieHandler, uploadHandler *handlers.UploadHandler) {
	// API versioning
	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Movie routes - CRUD operations
	movies := v1.Group("/movies")
	{
		movies.Get("/", movieHandler.GetAllMovies)
		movies.Get("/:id", movieHandler.GetMovieByID)
		movies.Post("/", movieHandler.CreateMovie)
		movies.Put("/:id", movieHandler.UpdateMovie)
		movies.Delete("/:id", movieHandler.DeleteMovie)
	}

	// Sync routes - TMDB synchronization
	sync := v1.Group("/sync")
	{
		sync.Post("/movies", movieHandler.SyncMoviesFromTMDB)
		sync.Get("/last-log", movieHandler.GetLastSyncLog)
	}

	// Dashboard routes - Analytics and statistics
	dashboard := v1.Group("/dashboard")
	{
		dashboard.Get("/stats", movieHandler.GetDashboardStats)
	}

	// Chart routes - Visualization data
	charts := v1.Group("/charts")
	{
		charts.Get("/", movieHandler.GetChartData)
		charts.Get("/pie", movieHandler.GetPieChartData)
		charts.Get("/column", movieHandler.GetColumnChartData)
		charts.Get("/monthly/:year", movieHandler.GetMonthlyChartData)
	}

	upload := v1.Group("/upload")
	{
		upload.Get("/presign", uploadHandler.GetPresignedURL)
	}
}
