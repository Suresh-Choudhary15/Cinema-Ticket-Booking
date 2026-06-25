# Cinema Ticket Booking

Live: https://cinema-ticket-booking-production.up.railway.app/

A backend service for booking cinema seats, built in Go and backed by Redis. The core problem this project solves is **preventing double-booking when multiple users try to claim the same seat at the same time** — the booking flow uses Redis's atomic `SETNX` operation as a distributed lock, so only one request can ever successfully hold a given seat, even under heavy concurrent load.

## How It Works

A seat reservation happens in two steps:

1. **Hold** — a user requests a seat, which creates a temporary reservation (a "session") with a 2-minute TTL. If the seat is already held or confirmed, the request fails immediately with `seat is already taken`.
2. **Confirm or Release** — within the hold window, the user either confirms the booking (making it permanent, no TTL) or releases it (freeing the seat for others). If neither happens, the hold expires automatically after 2 minutes and the seat becomes available again.

Seat state is stored in Redis using two key patterns:

- `seat:{movieID}:{seatID}` → the session data (JSON), with a TTL while held and no TTL once confirmed
- `session:{sessionID}` → a reverse lookup pointing back to the seat key

## Tech Stack

- **Go** (module: `github.com/sikozonpc/cinema`, Go 1.25)
- **Redis** — via `github.com/redis/go-redis/v9`
- **Docker Compose** — for running Redis and Redis Commander locally
- **Vanilla HTML/CSS/JS** — no frontend framework, no build step
- `github.com/google/uuid` — for generating session/booking IDs

## Project Structure

```
cmd/main.go                          → HTTP server entry point, route definitions
internal/booking/domain.go           → Booking struct and BookingStore interface
internal/booking/service.go          → Business logic layer (delegates to a store)
internal/booking/handler.go          → HTTP handlers (request/response handling)
internal/booking/redis_store.go      → Redis-backed implementation of BookingStore
internal/booking/memory_store.go     → In-memory store (not wired into main.go)
internal/booking/concurrent_store.go → Mutex-protected in-memory store (not wired into main.go)
internal/booking/service_test.go     → Concurrency test (100,000 simultaneous booking attempts)
internal/adapters/redis/redis.go     → Redis client constructor
internal/utils/utils.go              → Shared JSON response helper
static/index.html                    → Frontend: movie list, seat grid, checkout, countdown timer
docker-compose.yaml                  → Redis + Redis Commander services
```

> Note: `memory_store.go` and `concurrent_store.go` are present in the codebase but are not used by `cmd/main.go`, which wires up `RedisStore` exclusively. Their `Book()` method signatures also no longer match the current `BookingStore` interface.

## API Endpoints

| Method   | Path                                    | Description                                                  |
| -------- | --------------------------------------- | ------------------------------------------------------------ |
| `GET`    | `/movies`                               | List available movies (hardcoded: Inception, Dune: Part Two) |
| `GET`    | `/movies/{movieID}/seats`               | List seat statuses (booked/confirmed) for a movie            |
| `POST`   | `/movies/{movieID}/seats/{seatID}/hold` | Hold a seat, creating a 2-minute session                     |
| `PUT`    | `/sessions/{sessionID}/confirm`         | Confirm a held session, making the booking permanent         |
| `DELETE` | `/sessions/{sessionID}`                 | Release a held session, freeing the seat                     |
| `GET`    | `/`                                     | Serves the frontend from `static/`                           |

## Running Locally

**Prerequisites:** Go 1.25+, Docker

```bash
# 1. Start Redis and Redis Commander
docker-compose up -d

# 2. Download dependencies
go mod download

# 3. Run the server
go run cmd/main.go
```

The app runs on **http://localhost:8080**. Redis Commander (a GUI for inspecting Redis keys) is available at **http://localhost:8081**.

## Running the Concurrency Test

The test suite includes a load test that spawns 100,000 concurrent goroutines, all attempting to book the same seat at the same time, and asserts that exactly one succeeds:

```bash
go test ./internal/booking/... -run TestConcurrentBooking_ExactlyOneWins -v
```

This requires Redis to be running (`docker-compose up -d`), since it exercises the real Redis-backed store.

## Frontend

`static/index.html` is a single self-contained file (no build tooling) that:

- Generates a random anonymous user ID per browser session
- Renders a seat grid for the selected movie
- Polls `/movies/{movieID}/seats` every 2 seconds to reflect other users' holds/bookings
- Shows a live countdown timer for the current hold, with a confirm/release checkout panel
