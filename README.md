# Prosig - Blogging Platform API

A RESTful API for managing blog posts and comments, built with Go's standard library.

## Features

- RESTful API endpoints for managing blog posts and comments
- Request logging middleware using `slog`
- Basic authentication with session management
- Prometheus metrics for monitoring
- Docker support with multi-stage builds
- In-memory data storage (thread-safe)

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (optional, for containerized deployment)

## Running Locally

### Using Go directly:

1. Install dependencies:
```bash
go mod download
```

1. Run the application:
```bash
go run .
```

The API will be available at `http://localhost:8080`

### Environment Variables

- `PORT`: Server port (default: 8080)
- `AUTH_USERNAME`: Basic auth username (default: admin)
- `AUTH_PASSWORD`: Basic auth password (default: secret)

Example:
```bash
PORT=8080 AUTH_USERNAME=admin AUTH_PASSWORD=secret go run .
```

## Running with Docker

### Using Docker Compose (recommended):

```bash
docker-compose up --build
```

### Using Docker directly:

```bash
docker build -t prosig .
docker run -p 8080:8080 -e AUTH_USERNAME=admin -e AUTH_PASSWORD=secret prosig
```

## API Endpoints

All endpoints require Basic Authentication. After the first successful authentication, a session cookie is set for subsequent requests.

### Health Check

#### GET /health
Returns service health status.

**Response:**
```json
{
  "status": "healthy",
  "time": "2024-01-01T12:00:00Z",
  "request_id": "abc123...",
  "service": {
    "name": "prosig",
    "version": "1.0.0"
  }
}
```

### Authentication

Use Basic Authentication with the credentials set via environment variables:
- Username: `AUTH_USERNAME` (default: admin)
- Password: `AUTH_PASSWORD` (default: secret)

### Endpoints

#### GET /api/posts
Returns a list of all blog posts with comment counts.

**Response:**
```json
[
  {
    "id": 1,
    "title": "My First Post",
    "created_at": "2024-01-01T00:00:00Z",
    "comment_count": 2
  }
]
```

#### POST /api/posts
Creates a new blog post.

**Request Body:**
```json
{
  "title": "My Post Title",
  "content": "Post content here"
}
```

**Response:**
```json
{
  "id": 1,
  "title": "My Post Title",
  "content": "Post content here",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### GET /api/posts/{id}
Retrieves a specific blog post with all its comments.

**Response:**
```json
{
  "id": 1,
  "title": "My Post Title",
  "content": "Post content here",
  "created_at": "2024-01-01T00:00:00Z",
  "comments": [
    {
      "id": 1,
      "post_id": 1,
      "content": "Great post!",
      "created_at": "2024-01-01T01:00:00Z"
    }
  ]
}
```

#### POST /api/posts/{id}/comments
Adds a new comment to a blog post.

**Request Body:**
```json
{
  "content": "This is a comment"
}
```

**Response:**
```json
{
  "id": 1,
  "post_id": 1,
  "content": "This is a comment",
  "created_at": "2024-01-01T01:00:00Z"
}
```

#### GET /metrics
Prometheus metrics endpoint (no authentication required).

## Testing

Example using curl:

```bash
# Create a post
curl -u admin:secret -X POST http://localhost:8080/api/posts \
  -H "Content-Type: application/json" \
  -d '{"title":"Test Post","content":"This is a test"}'

# List posts
curl -u admin:secret http://localhost:8080/api/posts

# Get specific post
curl -u admin:secret http://localhost:8080/api/posts/1

# Add comment
curl -u admin:secret -X POST http://localhost:8080/api/posts/1/comments \
  -H "Content-Type: application/json" \
  -d '{"content":"Nice post!"}'
```
