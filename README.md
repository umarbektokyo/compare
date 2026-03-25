# Compare

A head-to-head comparison platform where users create rooms, add items, and vote on pairwise matchups. Items are ranked using a Glicko-inspired rating system.

## Features

- **Rooms** -- Create themed comparison rooms (e.g. "Best Programming Language").
- **Leaderboard** -- Items ranked by Elo with rating deviation (confidence interval).
- **Play mode** -- Vote on random X vs Y matchups with animated results.
- **Glicko ratings** -- Rating deviation tracks confidence; head-to-head dampening prevents repeated-matchup abuse.
- **Auth** -- Username/password with bcrypt hashing and JWT sessions. Voting is anonymous; auth controls room/item management.
- **Images** -- Optional image URLs on items.

## Tech Stack

| Layer    | Technology                        |
|----------|-----------------------------------|
| Backend  | Go + SQLite (via go-sqlite3)      |
| Frontend | SvelteKit (static adapter)        |
| Auth     | bcrypt + JWT (HS256, 30-day TTL)  |
| Deploy   | Single Docker container           |

## Quick Start

### Docker (recommended)

```bash
docker compose up --build
# Visit http://localhost:8080
```

Data is persisted in a Docker volume (`compare-data`).

### Local Development

**Prerequisites:** Go 1.24+, Node.js 22+

```bash
# Backend (serves frontend + API on http://localhost:8080)
cd backend && go run .

# Frontend dev (optional, for hot reload)
cd frontend && npm install && npm run dev
# Set VITE_API_URL=http://localhost:8080/api in .env if using separate dev server
```

### Build from Source

```bash
cd frontend && npm ci && npm run build && cd ..
cd backend && go build -o compare-server . && ./compare-server
```

## Project Structure

```
compare/
  Dockerfile         -- Multi-stage build (node -> go -> alpine)
  docker-compose.yml -- Single-service compose file
  backend/
    main.go          -- Initialization, DB schema, routing, HTTP helpers
    auth.go          -- JWT generation/validation, register/login handlers
    handlers.go      -- Room, item, match, and vote API handlers
    glicko.go        -- Glicko rating algorithm implementation
    glicko_test.go   -- Rating system unit tests
    handlers_test.go -- API handler integration tests
  frontend/
    src/
      lib/
        api.ts            -- Typed API client
        auth.svelte.ts    -- Reactive auth state (Svelte 5 runes)
      routes/
        +layout.svelte    -- App shell, navbar, auth modal
        +page.svelte      -- Home: room list
        room/[id]/
          +page.svelte    -- Room: leaderboard + history
          play/+page.svelte -- Play: X vs Y voting arena
```

## API Reference

All endpoints are prefixed with `/api`. JSON request/response bodies.

### Auth

| Method | Path                 | Auth | Description                |
|--------|----------------------|------|----------------------------|
| POST   | `/auth/register`     | No   | Create account             |
| POST   | `/auth/login`        | No   | Log in, receive JWT        |
| GET    | `/auth/me`           | Yes  | Get current user info      |

### Rooms

| Method | Path            | Auth  | Description              |
|--------|-----------------|-------|--------------------------|
| GET    | `/rooms`        | No    | List all rooms           |
| POST   | `/rooms`        | Yes   | Create a room            |
| GET    | `/rooms/:id`    | No    | Get room details         |
| DELETE | `/rooms/:id`    | Owner | Delete room + all data   |

### Items

| Method | Path                  | Auth  | Description           |
|--------|-----------------------|-------|-----------------------|
| GET    | `/rooms/:id/items`    | No    | List items (by Elo)   |
| POST   | `/rooms/:id/items`    | Owner | Add an item           |
| DELETE | `/items/:id`          | Owner | Remove an item        |

### Play

| Method | Path                  | Auth | Description                    |
|--------|-----------------------|------|--------------------------------|
| GET    | `/rooms/:id/pair`     | No   | Get random pair for voting     |
| POST   | `/vote`               | No   | Submit vote, returns new Elos  |
| GET    | `/rooms/:id/history`  | No   | Last 50 matches                |

## Rating System

Based on Mark Glickman's Glicko system:

- **Rating** (default 1500): estimated strength.
- **RD** (Rating Deviation, default 350): uncertainty. Decreases toward 50 as matches accumulate.
- **g(RD)**: weights opponent's rating by their certainty.
- **Expected score**: `E = 1 / (1 + 10^(-g(RD_opp) * (R - R_opp) / 400))`
- **H2H dampening**: `factor = 1 / (1 + 0.15 * prior_matchup_count)` -- repeated matchups yield diminishing rating changes.
- **Rating floor**: no item drops below 100.

## Environment Variables

| Variable     | Default     | Description                                          |
|--------------|-------------|------------------------------------------------------|
| `PORT`       | `8080`      | HTTP server port                                     |
| `DATA_DIR`   | `.`         | Directory for SQLite DB and auto-generated secret    |
| `JWT_SECRET` | *(auto)*    | JWT signing key. Set for production; auto-generates if empty |

## Running Tests

```bash
cd backend && go test -v ./...
```

## License

MIT
