# Movie Backend API

REST API for movie management application, built with Go Fiber, PostgreSQL, and TMDB API.

## Tech Stack

- **Go Fiber v2** - Web framework
- **PostgreSQL** - Database  
- **GORM** - ORM
- **TMDB API** - Movie data
- **MinIO** - Image storage
- **Swagger** - API docs

## Features

- ✅ CRUD Movies with pagination & search
- ✅ Sync data from TMDB API
- ✅ Upload poster images
- ✅ Dashboard analytics
- ✅ Filter & sorting movies
- ✅ Auto database migration

## Prerequisites

- Go 1.24+
- PostgreSQL 12+
- TMDB API Key (register at [TMDB](https://www.themoviedb.org/settings/api))

## Setup

### 1. Clone & Install

```bash
git clone <repository-url>
cd movie-backend
go mod download
```

### 2. Environment Variables

Create `envs/.env` file:

```env
# Server
SERVER_PORT=8010

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=movie_db
DB_SSLMODE=disable

# TMDB API
TMDB_API_KEY=your_tmdb_api_key_here
TMDB_BASE_URL=https://api.themoviedb.org/3

# MinIO/S3 (optional)
AWS_ENDPOINT=storage.example.com
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_BUCKET=movies
AWS_DEFAULT_REGION=us-east-1
```

### 3. Build & Run

```bash
go build -o bin/movie-backend cmd/main.go
./bin/movie-backend
```

Server runs at `http://localhost:8010`

## API Endpoints

### Health Check
```
GET /health
```

### Movies
```
GET    /api/v1/movies          # List movies
GET    /api/v1/movies/:id      # Get movie
POST   /api/v1/movies          # Create movie
PUT    /api/v1/movies/:id      # Update movie
DELETE /api/v1/movies/:id      # Delete movie
```

**Query Parameters:**
- `page` (default: 1): Page number
- `limit` (default: 20): Items per page
- `search`: Search by title/overview
- `sort_by`: Sort field (vote_average, popularity, etc.)
- `order`: ASC or DESC
- `genre_id`: Filter by genre
- `min_rating`: Minimum rating
- `year`: Filter by release year

### Sync
```
POST /api/v1/sync/movies?pages=5    # Sync from TMDB
GET  /api/v1/sync/last-log          # Last sync log
```

### Dashboard
```
GET /api/v1/dashboard/stats         # Dashboard statistics
```

### Upload
```
POST /api/v1/upload/poster          # Upload poster image
```

### Master Data
```
GET /api/v1/genres                  # List genres
GET /api/v1/languages               # List languages
```

## Swagger Documentation

```
http://localhost:8010/swagger/index.html
```

## Project Structure

```
movie-backend/
├── cmd/
│   └── main.go                # Entry point
├── internal/
│   ├── config/               # Config management
│   ├── handlers/             # HTTP handlers
│   ├── models/              # Database models
│   ├── repository/          # Database operations
│   ├── routes/              # Route definitions
│   ├── services/            # Business logic
│   └── utils/               # Helper functions
├── docs/                    # Swagger docs
└── envs/                    # Environment files
```

## Development

### Hot Reload
```bash
go install github.com/cosmtrek/air@latest
air
```

### Generate Swagger
```bash
swag init -g cmd/main.go -o docs
```

### Run Tests
```bash
go test ./...
```

## Docker

```bash
# Build image
docker build -f docker/Dockerfile -t movie-backend:latest .

# Run container
docker run -p 8010:8010 --env-file envs/.env movie-backend:latest
```

## Troubleshooting

### Database Connection Issues
- Ensure PostgreSQL is running
- Verify database credentials in `.env` file
- Create database if needed: `createdb -U postgres movie_db`

### TMDB API Issues
- Verify API key is valid at [TMDB Settings](https://www.themoviedb.org/settings/api)
- Check API rate limits

### Port Already in Use
```bash
# Find process using port 8010
lsof -i :8010

# Kill the process
kill -9 <PID>
```

---

**Built with ❤️ using Go + Fiber + PostgreSQL**
