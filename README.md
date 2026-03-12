# Shipyard

Shipyard is an MVP deployment orchestration platform for static sites.
It accepts a Git repository + build config, runs a containerized build, uploads artifacts to MinIO (S3-compatible), and serves deployments by ID.

## Features

- Deployment API (`POST /deploy`)
- FIFO worker with retry logic
- Docker-based build execution
- MinIO artifact storage
- Deployment status endpoint
- Deployment logs endpoint
- NGINX reverse proxy
- Artifact checksum persistence (`artifact_checksum`)

## Tech Stack

- Go (Echo v5)
- PostgreSQL
- MinIO
- Docker / Docker Compose
- NGINX

## Architecture

1. API receives deployment request and inserts a `QUEUED` deployment row.
2. Worker polls queue, transitions to `BUILDING`, and runs the build in Docker.
3. Built files are uploaded to MinIO bucket `deployments` under `<deployment_id>/...`.
4. Worker stores artifact checksum and marks deployment as `READY`.
5. Static files are served via API/NGINX route using deployment ID.

## Prerequisites

- Docker
- Docker Compose

## Run With Docker Compose

```bash
docker compose up --build
```

This starts:

- `postgres` on `localhost:5432`
- `minio` API on `localhost:9000`
- `minio` console on `localhost:9001`
- `api` on `localhost:8082`
- `worker`
- `nginx` on `localhost:8001`

API key for protected endpoints (default in compose): `dev-shipyard-key`

## Database Setup (required)

Run this once after containers are up:

```bash
docker compose exec -T postgres psql -U postgres -d shipyard <<'SQL'
CREATE TABLE IF NOT EXISTS deployments (
  id UUID PRIMARY KEY,
  repo_url TEXT NOT NULL,
  build_command TEXT NOT NULL,
  output_dir TEXT NOT NULL,
  status TEXT NOT NULL,
  attempt_count INT NOT NULL DEFAULT 0,
  max_attempts INT NOT NULL DEFAULT 3,
  artifact_checksum TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS deployment_logs (
  id BIGSERIAL PRIMARY KEY,
  deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
  message TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
SQL
```

## API Endpoints

### Health

```bash
curl http://localhost:8082/healthz
```

### Create Deployment

```bash
curl -X POST http://localhost:8082/deploy \
  -H "X-API-Key: dev-shipyard-key" \
  -H "Content-Type: application/json" \
  -d '{
    "repo_url":"https://github.com/<owner>/<repo>",
    "build_command":"npm install && npm run build",
    "output_dir":"dist"
  }'
```

Response includes `deployment_id`.

### Get Deployment Status

```bash
curl http://localhost:8082/deployments/<deployment_id>
# add auth header:
# -H "X-API-Key: dev-shipyard-key"
```

### Get Deployment Logs

```bash
curl http://localhost:8082/logs/<deployment_id>
# add auth header:
# -H "X-API-Key: dev-shipyard-key"
```

### Serve Deployment

- Via API:
  - `http://localhost:8082/<deployment_id>`
  - `http://localhost:8082/<deployment_id>/assets/app.js`
- Via NGINX proxy:
  - `http://localhost:8001/<deployment_id>`

## MinIO

- Console: `http://localhost:9001`
- Username: `minioadmin`
- Password: `minioadmin`

Artifacts are stored in bucket `deployments`.

## Important Notes

1. `POST /deploy`, `GET /deployments/:id`, and `GET /logs/:id` now require an API key:
   - Header `X-API-Key: <key>` or `Authorization: Bearer <key>`
   - Set via env var `SHIPYARD_API_KEY`

2. `nginx.conf` currently proxies to `http://172.17.0.1:8080`, but API listens on `8082`.
Change proxy target to `http://172.17.0.1:8082` (or service DNS `http://api:8082`) for correct routing.

3. Worker runs `docker run` internally for builds. If worker is containerized, ensure it can access Docker:
   - mount Docker socket: `/var/run/docker.sock:/var/run/docker.sock`
   - have Docker CLI available in worker image

Without this, build jobs may fail inside the worker container.

## Useful Commands

```bash
# Rebuild and restart

docker compose up --build

# Follow API logs

docker compose logs -f api

# Follow worker logs

docker compose logs -f worker

# Stop all

docker compose down
```
