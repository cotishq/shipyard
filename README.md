# Shipyard

Shipyard is an MVP deployment orchestration platform for static sites.
It uses project-level configuration (repo + build settings), runs containerized builds, uploads artifacts to MinIO (S3-compatible), and serves deployments by ID.

## Features

- Project APIs (`POST /projects`, `GET /projects`, `GET /projects/:id`)
- Deployment trigger API (`POST /projects/:id/deployments`)
- FIFO worker with retry logic
- Docker-based build execution
- MinIO artifact storage
- Deployment status endpoint
- Deployment logs endpoint
- NGINX reverse proxy
- Artifact checksum persistence (`artifact_checksum`)
- GitHub Webhooks for automated deployments
- Deployment actions: retry, cancel, and redeploy

## Tech Stack

- Go (Echo v5)
- PostgreSQL
- MinIO
- Docker / Docker Compose
- NGINX

## Architecture

1. API receives project deployment trigger and inserts a `QUEUED` deployment row linked to `project_id`.
2. Worker polls queue, transitions to `BUILDING`, and runs the build in Docker.
3. Built files are uploaded to MinIO bucket `deployments` under `<deployment_id>/...`.
4. Worker stores artifact checksum and marks deployment as `READY`.
5. Static files are served via API/NGINX route using deployment ID.

## How Shipyard Works

1. A user creates a project with `repo_url`, `build_preset`, `output_dir`, and `default_branch`.
2. A deployment is triggered for that project via `POST /projects/:id/deployments`.
3. Shipyard inserts a `QUEUED` deployment row linked to the project.
4. The worker clones the configured repo/branch, runs the preset build in Docker, and uploads artifact output to MinIO.
5. Shipyard records deployment logs, lifecycle metadata, and artifact checksum.
6. If the build succeeds, deployment becomes `READY` and is served at `/<deployment_id>`.
7. Users inspect status with `GET /deployments/:id`, list history with `GET /deployments`, and read logs with `GET /logs/:id`.
8. Users can retry failed deployments, cancel queued/building deployments, or redeploy from any completed/failed deployment.

### Deployment Statuses

- `QUEUED` - Deployment is waiting in the queue
- `BUILDING` - Worker is currently building the deployment
- `READY` - Deployment was built successfully and is being served
- `FAILED` - Deployment failed (can be retried)
- `CANCELLED` - Deployment was cancelled by user

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

Token auth is required for protected endpoints (`X-API-Key` or `Authorization: Bearer <token>`).

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
- `PROJECT_NAME`
- `HEALTH_TIMEOUT_SECONDS`
- `DEPLOY_TIMEOUT_SECONDS`
- `POLL_INTERVAL_SECONDS`

### Create Project

```bash
curl -X POST http://localhost:8082/projects \
  -H "X-API-Key: <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"my-site",
    "repo_url":"https://github.com/<owner>/<repo>",
    "build_preset":"vite",
    "output_dir":"dist",
    "default_branch":"main"
  }'
```

Response includes `project_id`.

#### Request fields

- **`name`**: required. Unique per user.
- **`repo_url`**: required. Must be a valid `https` URL. Whitespace is rejected.
- **`build_preset`**: required. One of the supported presets listed below.
- **`output_dir`**: required for most presets. Must be a relative path inside the repo (no leading `/`, no `..` traversal). If omitted for `static-copy`, the repo contents are copied to the artifact workspace.
- **`default_branch`**: optional. Defaults to `main`.

#### Supported `build_preset` values

- **`static-copy`**: copies files without building (useful for already-built/static repos)
- **`npm`**: runs `npm ci && npm run build`
- **`vite`**: runs `npm ci && npm run build`
- **`next-export`**: runs `npm ci && npm run build && npm run export`

#### Repo host allowlist

For safety, Shipyard currently only allows cloning from:

- `github.com`

### Trigger Deployment For Project

```bash
curl -X POST http://localhost:8082/projects/<project_id>/deployments \
  -H "X-API-Key: <token>"
```

Response includes `deployment_id`.

### Get Deployment Status

```bash
curl http://localhost:8082/deployments/<deployment_id> \
  -H "X-API-Key: <token>"
```

Response now includes lifecycle metadata:
- `started_at`
- `finished_at`
- `error_message`
- `build_duration_seconds`

### List Deployments

```bash
curl "http://localhost:8082/deployments?limit=20&offset=0" \
  -H "X-API-Key: <token>"
```

### Get Deployment Logs

```bash
curl http://localhost:8082/logs/<deployment_id> \
  -H "X-API-Key: <token>"
```

### Retry Deployment

Retries a failed deployment by resetting it back to the queue with attempt count reset to 0.

```bash
curl -X POST http://localhost:8082/deployments/<deployment_id>/retry \
  -H "X-API-Key: <token>"
```

Response includes `deployment_id`. Only failed deployments can be retried.

### Cancel Deployment

Cancels a queued or building deployment.

```bash
curl -X POST http://localhost:8082/deployments/<deployment_id>/cancel \
  -H "X-API-Key: <token>"
```

Response includes `deployment_id`. Only queued or building deployments can be cancelled.

### Redeploy Deployment

Creates a new deployment based on an existing deployment's configuration.

```bash
curl -X POST http://localhost:8082/deployments/<deployment_id>/redeploy \
  -H "X-API-Key: <token>"
```

Response includes the new `deployment_id`. Can be used on ready, failed, or cancelled deployments.

### List Projects

```bash
curl http://localhost:8082/projects \
  -H "X-API-Key: <token>"
```

### Serve Deployment

- Via API:
  - `http://localhost:8082/<deployment_id>`
  - `http://localhost:8082/<deployment_id>/assets/app.js`
- Via NGINX proxy:
  - `http://localhost:8001/<deployment_id>`

### Create API Token

```bash
curl -X POST http://localhost:8082/tokens \
  -H "X-API-Key: <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-token","expires_at":"2025-12-31T23:59:59Z"}'
```

Response includes the raw token (shown only once).

### List API Tokens

```bash
curl http://localhost:8082/tokens \
  -H "X-API-Key: <token>"
```

### Revoke API Token

```bash
curl -X DELETE http://localhost:8082/tokens/<token_id> \
  -H "X-API-Key: <token>"
```

## Webhooks

Shipyard supports GitHub webhooks for automated deployments. Each project can have its own webhook URL.

### Create Project Webhook

```bash
curl -X POST http://localhost:8082/projects/<project_id>/webhook \
  -H "X-API-Key: <token>"
```

Response includes `webhook_url` (e.g., `http://localhost:8082/webhooks/github?project_id=<project_id>&secret=<secret>`).

### Get Project Webhook

```bash
curl http://localhost:8082/projects/<project_id>/webhook \
  -H "X-API-Key: <token>"
```

Returns the webhook URL and secret. The secret is used to verify webhook payloads.

### GitHub Webhook Configuration

In your GitHub repository settings:

1. Go to **Settings > Webhooks > Add webhook**
2. **Payload URL**: Use the webhook URL from above
3. **Content type**: `application/json`
4. **Events**: Select "Just the push event" (or customize as needed)
5. **Secret**: Enter the secret from the webhook response
6. **Add webhook**

When a push event occurs, GitHub sends a webhook to Shipyard, which automatically triggers a new deployment for the configured branch.

## MinIO

- Console: `http://localhost:9001`
- Username: `shipyard_minio`
- Password: `shipyard_minio_change_me`

Artifacts are stored in bucket `deployments`.

## Important Notes

1. `POST /projects`, `GET /projects`, `GET /projects/:id`, `POST /projects/:id/deployments`, `GET /deployments`, `GET /deployments/:id`, and `GET /logs/:id` require token auth:
   - Header `X-API-Key: <key>` or `Authorization: Bearer <key>`
   - Tokens are stored hashed in `api_tokens`
   - Raw tokens are never stored in DB

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
