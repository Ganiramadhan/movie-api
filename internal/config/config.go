package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	TMDB     TMDBConfig
	MinIO    MinIOConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	QueryTimeout    time.Duration
}

type TMDBConfig struct {
	APIKey      string
	BaseURL     string
	HTTPTimeout time.Duration
}

type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Region          string
	UseSSL          bool
	PublicURL       string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnvOrDefault("SERVER_PORT", "8010"),
			ReadTimeout:  getDurationOrDefault("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDurationOrDefault("SERVER_WRITE_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnvOrDefault("DB_HOST", "localhost"),
			Port:            getEnvOrDefault("DB_PORT", "5432"),
			User:            getEnvOrDefault("DB_USER", "postgres"),
			Password:        getEnvOrDefault("DB_PASSWORD", "postgres"),
			DBName:          getEnvOrDefault("DB_NAME", "movie_db"),
			SSLMode:         getEnvOrDefault("DB_SSLMODE", "disable"),
			MaxOpenConns:    getIntOrDefault("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntOrDefault("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationOrDefault("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			QueryTimeout:    getDurationOrDefault("DB_QUERY_TIMEOUT", 10*time.Second),
		},
		TMDB: TMDBConfig{
			APIKey:      os.Getenv("TMDB_API_KEY"),
			BaseURL:     getEnvOrDefault("TMDB_BASE_URL", "https://api.themoviedb.org/3"),
			HTTPTimeout: getDurationOrDefault("TMDB_HTTP_TIMEOUT", 30*time.Second),
		},
		MinIO: MinIOConfig{
			Endpoint:        getEnvOrDefault("AWS_ENDPOINT", "storage.bpdabujapijabar.or.id"),
			AccessKeyID:     getEnvOrDefault("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnvOrDefault("AWS_SECRET_ACCESS_KEY", ""),
			BucketName:      getEnvOrDefault("AWS_BUCKET", "movies"),
			Region:          getEnvOrDefault("AWS_DEFAULT_REGION", "us-east-1"),
			UseSSL:          getBoolOrDefault("AWS_USE_SSL", true), // Use SSL by default for HTTPS
			PublicURL:       getEnvOrDefault("AWS_URL", "https://storage.bpdabujapijabar.or.id/movies"),
		},
	}
}

// GetDSN returns PostgreSQL connection string
func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

func (c *Config) Validate() error {
	if c.TMDB.APIKey == "" {
		return fmt.Errorf("TMDB_API_KEY is required")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.MinIO.AccessKeyID == "" {
		return fmt.Errorf("AWS_ACCESS_KEY_ID is required for MinIO")
	}
	if c.MinIO.SecretAccessKey == "" {
		return fmt.Errorf("AWS_SECRET_ACCESS_KEY is required for MinIO")
	}
	if c.MinIO.Endpoint == "" {
		return fmt.Errorf("AWS_ENDPOINT is required for MinIO")
	}
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
