# Shipyard

Shipyard is an MVP deployment orchestration platform for static sites.
It accepts a Git repository + build preset, runs a containerized build, uploads artifacts to MinIO (S3-compatible), and serves deployments by ID.

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

## How Shipyard Works

1. A user submits a GitHub repository, `build_preset`, and `output_dir` to `POST /deploy`.
2. Shipyard stores the deployment in Postgres with status `QUEUED`.
3. The worker picks up the job, clones the repo, and runs the preset build in Docker.
4. Only the built artifact output is copied into the deployment workspace and uploaded to MinIO.
5. Shipyard records deployment logs, lifecycle metadata, and an artifact checksum.
6. If the build succeeds, the deployment becomes `READY` and is served at `/<deployment_id>`.
7. Users can inspect status with `GET /deployments/:id`, list history with `GET /deployments`, and read logs with `GET /logs/:id`.

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

API key for protected endpoints (from compose example): `shipyard_api_key_change_me_please`

## Database Setup

Database migrations are applied automatically on startup by both `api` and `worker`.
Migration files live in `migrations/` and are tracked in the `schema_migrations` table.

For CI or local verification, starting the app is enough to apply pending migrations:

```bash
docker compose up --build
```

## API Endpoints

### Health

```bash
curl http://localhost:8082/healthz
```

### Smoke Test

```bash
bash scripts/smoke_test.sh
```

Optional overrides:
- `API_URL`
- `PROXY_URL`
- `API_KEY`
- `HEALTH_TIMEOUT_SECONDS`
- `DEPLOY_TIMEOUT_SECONDS`
- `POLL_INTERVAL_SECONDS`

### Create Deployment

```bash
curl -X POST http://localhost:8082/deploy \
  -H "X-API-Key: shipyard_api_key_change_me_please" \
  -H "Content-Type: application/json" \
  -d '{
    "repo_url":"https://github.com/<owner>/<repo>",
    "build_preset":"vite",
    "output_dir":"dist"
  }'
```

Response includes `deployment_id`.

#### Request fields

- **`repo_url`**: required. Must be a valid `https` URL. Whitespace is rejected.
- **`build_preset`**: required. One of the supported presets listed below.
- **`output_dir`**: required for most presets. Must be a relative path inside the repo (no leading `/`, no `..` traversal). If omitted for `static-copy`, the repo contents are copied to the artifact workspace.

#### Supported `build_preset` values

- **`static-copy`**: copies files without building (useful for already-built/static repos)
- **`npm`**: runs `npm ci && npm run build`
- **`vite`**: runs `npm ci && npm run build`
- **`next-export`**: runs `npm ci && npm run build && npm run export`

#### Repo host allowlist

For safety, Shipyard currently only allows cloning from:

- `github.com`

### Get Deployment Status

```bash
curl http://localhost:8082/deployments/<deployment_id> \
  -H "X-API-Key: shipyard_api_key_change_me_please"
```

Response now includes lifecycle metadata:
- `started_at`
- `finished_at`
- `error_message`
- `build_duration_seconds`

### List Deployments

```bash
curl "http://localhost:8082/deployments?limit=20&offset=0" \
  -H "X-API-Key: shipyard_api_key_change_me_please"
```

### Get Deployment Logs

```bash
curl http://localhost:8082/logs/<deployment_id> \
  -H "X-API-Key: shipyard_api_key_change_me_please"
```

### Serve Deployment

- Via API:
  - `http://localhost:8082/<deployment_id>`
  - `http://localhost:8082/<deployment_id>/assets/app.js`
- Via NGINX proxy:
  - `http://localhost:8001/<deployment_id>`

## MinIO

- Console: `http://localhost:9001`
- Username: `shipyard_minio`
- Password: `shipyard_minio_change_me`

Artifacts are stored in bucket `deployments`.

## Important Notes

1. `POST /deploy`, `GET /deployments`, `GET /deployments/:id`, and `GET /logs/:id` require an API key:
   - Header `X-API-Key: <key>` or `Authorization: Bearer <key>`
   - Set via env var `SHIPYARD_API_KEY`
   - Default development key values are blocked unless `SHIPYARD_ALLOW_INSECURE_DEFAULTS=true`

2. `nginx.conf` proxies to `http://api:8082` inside Docker Compose network.
If you run NGINX outside Compose, adjust upstream accordingly.

3. Worker runs `docker run` internally for builds. If worker is containerized, ensure it can access Docker:
   - mount Docker socket: `/var/run/docker.sock:/var/run/docker.sock`
   - have Docker CLI available in worker image

Without this, build jobs may fail inside the worker container.

## Useful Commands

```bash
# Rebuild and restart

docker compose up --build

# Inspect applied migrations

docker compose exec postgres psql -U shipyard -d shipyard -c "SELECT * FROM schema_migrations ORDER BY applied_at;"

# Follow API logs

docker compose logs -f api

# Follow worker logs

docker compose logs -f worker

# Stop all

docker compose down
```
