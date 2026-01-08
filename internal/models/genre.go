package models

import "time"

type Genre struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TMDBID    int       `gorm:"uniqueIndex;not null" json:"tmdb_id"`
	Name      string    `gorm:"not null;index" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Genre) TableName() string {
	return "genres"
}

type MovieGenre struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MovieID   uint      `gorm:"index;not null" json:"movie_id"`
	GenreID   uint      `gorm:"index;not null" json:"genre_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (MovieGenre) TableName() string {
	return "movie_genres"
}
