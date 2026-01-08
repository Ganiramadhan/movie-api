package models

import (
	"time"
)

type Movie struct {
	ID            uint      `gorm:"primaryKey" json:"id" example:"1"`
	TMDBID        int       `gorm:"uniqueIndex;not null" json:"tmdb_id" example:"550"`
	Title         string    `gorm:"not null;index" json:"title" example:"Fight Club"`
	OriginalTitle string    `json:"original_title" example:"Fight Club"`
	Overview      string    `gorm:"type:text" json:"overview" example:"A ticking-Loss insurance clerk..."`
	ReleaseDate   string    `gorm:"index" json:"release_date" example:"1999-10-15"`
	PosterPath    string    `json:"poster_path" example:"/pB8BM7pdSp6B6Ih7QZ4DrQ3PmJK.jpg"`
	BackdropPath  string    `json:"backdrop_path" example:"/52AfXWuXCHn3UjD17rBruA9f5qb.jpg"`
	VoteAverage   float64   `gorm:"index" json:"vote_average" example:"8.4"`
	VoteCount     int       `json:"vote_count" example:"26280"`
	Popularity    float64   `gorm:"index" json:"popularity" example:"61.416"`
	Adult         bool      `json:"adult" example:"false"`
	LanguageID    *uint     `gorm:"index" json:"language_id"`
	Language      *Language `gorm:"foreignKey:LanguageID" json:"language,omitempty"`
	Genres        []Genre   `gorm:"many2many:movie_genres;" json:"genres,omitempty"`
	CreatedAt     time.Time `gorm:"index" json:"created_at"`
	UpdatedAt     time.Time `gorm:"index" json:"updated_at"`
}

func (Movie) TableName() string {
	return "movies"
}

type TMDBMovieResponse struct {
	ID               int     `json:"id"`
	Title            string  `json:"title"`
	OriginalTitle    string  `json:"original_title"`
	Overview         string  `json:"overview"`
	ReleaseDate      string  `json:"release_date"`
	PosterPath       string  `json:"poster_path"`
	BackdropPath     string  `json:"backdrop_path"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	Popularity       float64 `json:"popularity"`
	Adult            bool    `json:"adult"`
	OriginalLanguage string  `json:"original_language"`
	GenreIDs         []int   `json:"genre_ids"`
}

type TMDBPopularMoviesResponse struct {
	Page         int                 `json:"page"`
	Results      []TMDBMovieResponse `json:"results"`
	TotalPages   int                 `json:"total_pages"`
	TotalResults int                 `json:"total_results"`
}

type SyncLog struct {
	ID            uint      `gorm:"primaryKey" json:"id" example:"1"`
	SyncType      string    `gorm:"index" json:"sync_type" example:"manual"`
	Status        string    `gorm:"index" json:"status" example:"success"`
	MoviesAdded   int       `json:"movies_added" example:"20"`
	MoviesUpdated int       `json:"movies_updated" example:"5"`
	ErrorMessage  string    `gorm:"type:text" json:"error_message,omitempty"`
	SyncedAt      time.Time `gorm:"index" json:"synced_at"`
	CreatedAt     time.Time `json:"created_at"`
}

func (SyncLog) TableName() string {
	return "sync_logs"
}

type DashboardStats struct {
	TotalMovies    int64      `json:"total_movies" example:"100"`
	AverageRating  float64    `json:"average_rating" example:"7.5"`
	TotalVotes     int64      `json:"total_votes" example:"500000"`
	LastSyncTime   *time.Time `json:"last_sync_time"`
	TopRatedMovies []Movie    `json:"top_rated_movies"`
	MostPopular    []Movie    `json:"most_popular"`
	RecentlyAdded  []Movie    `json:"recently_added"`
}

type PieChartData struct {
	Label string `json:"label" example:"English"`
	Value int64  `json:"value" example:"45"`
	Code  string `json:"code" example:"en"`
}

type ColumnChartData struct {
	Label string `json:"label" example:"2024"`
	Value int64  `json:"value" example:"15"`
}

type ChartDataResponse struct {
	PieChart    []PieChartData    `json:"pie_chart"`
	ColumnChart []ColumnChartData `json:"column_chart"`
}

type DateRangeFilter struct {
	StartDate string `json:"start_date" example:"2024-01-01"`
	EndDate   string `json:"end_date" example:"2024-12-31"`
}
