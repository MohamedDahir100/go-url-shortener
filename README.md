Go URL Shortener Microservice

A production-style URL shortening service built with Go, Gin, and Redis, and fully containerized with Docker and Docker Compose.
The service exposes RESTful endpoints to create short links and redirect users to the original URLs with low latency.

--------------------------------------------------
FEATURES

- RESTful API using Gin
  - POST /shorten to create short URLs
  - GET /:code to redirect to the original URL
- Unique short-code generation with collision handling
- Redis as a high-performance key–value store for persistence
- Stateless API design (state lives in Redis)
- Dockerized using a multi-stage build
- Docker Compose stack for running API + Redis together
- One-command startup for a reproducible dev environment

--------------------------------------------------
ARCHITECTURE (HIGH LEVEL)

Client (curl / browser)
        |
        v
   Go API (Gin)
        |
        v
      Redis

- The short code is the key
- The original URL is the value
- Redirect flow:
  1. Client sends a long URL to POST /shorten
  2. API generates a short code and stores it in Redis
  3. Client receives a short URL
  4. Client visits /:code
  5. API looks up the code in Redis and returns a 302 redirect

--------------------------------------------------
TECH STACK

- Go (Golang)
- Gin (HTTP web framework)
- Redis (key–value store)
- Docker (containerization)
- Docker Compose (multi-container orchestration)

--------------------------------------------------
PREREQUISITES

- Docker Desktop
- Git (optional)

--------------------------------------------------
RUN WITH DOCKER COMPOSE (RECOMMENDED)

From the project root:

docker compose up --build

This will:
- Build the Go API image
- Start Redis
- Start the API
- Expose the API on: http://localhost:8080

--------------------------------------------------
TEST THE API

1) Health Check

curl.exe http://localhost:8080/health

Expected response:
{"status":"ok","redis":true}

--------------------------------------------------
2) Create a Short URL (Windows-friendly)

Create a JSON file:

'{"original_url":"https://www.google.com"}' | Out-File -Encoding ascii body.json

Send request:

curl.exe -X POST http://localhost:8080/shorten -H "Content-Type: application/json" --data-binary "@body.json"

Example response:
{"short_url":"http://localhost:8080/p0fr4X4","code":"p0fr4X4"}

--------------------------------------------------
3) Redirect Using the Short Code

curl.exe -v http://localhost:8080/p0fr4X4

You should see:
- HTTP/1.1 302 Found
- Location: https://www.google.com

--------------------------------------------------
PERSISTENCE

Because Redis is used as the datastore:
- You can restart the Go container
- The short links will still work
- The API remains stateless and scalable

--------------------------------------------------
CONTAINERS

- app: Go API (Gin)
- redis: Redis 7 (Alpine)
- volume: redis_data for Redis persistence

Start/stop stack:

docker compose up --build
docker compose down

--------------------------------------------------
ENVIRONMENT VARIABLES

- PORT=8080
- BASE_URL=http://localhost:8080
- REDIS_HOST=redis
- REDIS_PORT=6379
- GIN_MODE=release (optional)

--------------------------------------------------
DESIGN NOTES

- Uses random, URL-safe short codes (~7 characters)
- Uses Redis SETNX for atomic collision prevention
- Keeps the API stateless for horizontal scaling
- Multi-stage Docker build produces a small runtime image
