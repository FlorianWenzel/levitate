# Levitate

> Open-source team scheduling and resource management. **They float. We levitate.**

Levitate is a self-hosted alternative to Float for tracking who's working on what, when. Drag projects across a workday-only grid, see capacity at a glance, manage time-off, export reports as CSV. Built as a single Go binary that serves both the API and the SPA.

![Schedule](docs/screenshots/dashboard.png)

[![CI](https://github.com/florianwenzel/levitate/actions/workflows/ci.yml/badge.svg)](https://github.com/florianwenzel/levitate/actions/workflows/ci.yml)
[![Image](https://github.com/florianwenzel/levitate/actions/workflows/docker.yml/badge.svg)](https://github.com/florianwenzel/levitate/actions/workflows/docker.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## Features

- **Drag-and-drop schedule grid** — workday-only columns (no weekends), bars stack flush when assignments overlap, side handles to resize duration, top handle to resize hours/day, right-click for Edit / Split / Delete.

  ![Schedule with stacked bars](docs/screenshots/schedule-stacking.png)

- **Click-and-drag to create assignments** with a live ghost selection that auto-clips around time-off.

  ![Drag to create](docs/screenshots/schedule-drag-to-create.png)

- **Time-off as a striped blocker** — full-row, diagonal-stripe overlay; assignments dragged or created across it are auto-clipped to the workday boundary.

  ![Time off](docs/screenshots/schedule-time-off.png)

- **Capacity heatmap** — per-person, per-week assigned vs. available hours, overallocation flagged in red. Workdays only — a Mon–Sun assignment at 8 h/day is 40 h/week, not 56.

  ![Capacity](docs/screenshots/capacity.png)

- **CSV exports** for utilization and assignments over any date range.

  ![Reports](docs/screenshots/reports.png)

- **OIDC auth out of the box** — bring your own provider (Keycloak, Auth0, Okta, etc.). Two roles: `admin` (full CRUD) and `member` (read-only). Users that sign in are automatically created as schedulable people.
- **Single binary, single image.** ~30 MB Alpine container. Bring your own Postgres + OIDC.

## Quick start (Docker)

You need a Postgres 16+ database and an OIDC provider already running somewhere. Then:

```bash
docker run -d --name levitate \
  -p 8080:8080 \
  -e LEVITATE_DB_URL='postgres://user:pass@db.example.com:5432/levitate?sslmode=require' \
  -e LEVITATE_OIDC_ISSUER='https://auth.example.com/realms/levitate' \
  -e LEVITATE_OIDC_AUDIENCE='levitate-frontend' \
  ghcr.io/florianwenzel/levitate:latest
```

Open http://localhost:8080. The container migrates the database on boot.

### OIDC client setup

Levitate is a public OIDC client (PKCE). In your provider, register a client with:

- **Client ID:** the same value you pass as `LEVITATE_OIDC_AUDIENCE` (e.g. `levitate-frontend`)
- **Client type:** public
- **Standard flow:** enabled, with PKCE S256
- **Valid redirect URI:** `https://your-levitate-host/callback`
- **Web origin:** `https://your-levitate-host`
- A claim mapper that puts the user's roles into a JSON-array claim (default name: `roles`). The roles `admin` and `member` are recognised.

A complete Keycloak realm export covering all of this lives in [`keycloak/realm-levitate.json`](keycloak/realm-levitate.json).

## Configuration

All settings are env-var driven (12-factor).

| Variable                       | Default                          | Description                                                                                                                                                                          |
| ------------------------------ | -------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `LEVITATE_HTTP_ADDR`           | `:8080`                          | Listen address.                                                                                                                                                                      |
| `LEVITATE_LOG_LEVEL`           | `info`                           | `debug` \| `info` \| `warn` \| `error`. Logs are emitted as structured JSON.                                                                                                         |
| `LEVITATE_DB_URL`              | _required_                       | Postgres connection URL. The binary runs migrations on boot.                                                                                                                         |
| `LEVITATE_OIDC_ISSUER`         | _required_                       | Public OIDC issuer URL — what the browser sees, e.g. `https://auth.example.com/realms/levitate`. Returned to the SPA via `/api/public/config`.                                       |
| `LEVITATE_OIDC_DISCOVERY_URL`  | same as `OIDC_ISSUER`            | URL the backend uses to fetch the OIDC discovery doc + JWKs. Set this only if your container reaches the IdP under a different hostname than the browser does (split-horizon DNS).   |
| `LEVITATE_OIDC_AUDIENCE`       | _empty (= no `aud` check)_       | Expected `aud` claim on access tokens. Doubles as the OIDC client ID returned to the SPA.                                                                                            |
| `LEVITATE_OIDC_ROLE_CLAIM`     | `roles`                          | Claim name carrying role strings. Supports dotted paths (e.g. `realm_access.roles`).                                                                                                 |
| `LEVITATE_CORS_ORIGINS`        | `*`                              | Comma-separated allowed origins. With the single-image deployment this isn't needed (SPA + API share an origin); set it if you front the API for an external SPA.                    |
| `LEVITATE_ALLOW_TEST_RESET`    | `false`                          | Enables `POST /api/test/reset`, used by the e2e suite. **Never enable in production.**                                                                                               |

## Architecture

```
                ┌──────────────────────────────────────────┐
                │           levitate (single image)        │
   browser ───▶ │  Go binary  ──▶  embedded Nuxt SPA       │
                │     │       ──▶  /api/* (chi + sqlc)     │
                │     │           │                        │
                └─────┼───────────┼────────────────────────┘
                      │           │
                      │           └─▶ Postgres 16+
                      └─▶ OIDC IdP (Keycloak, Auth0, …)
```

- **Backend:** Go 1.25, [chi](https://github.com/go-chi/chi) router, [sqlc](https://sqlc.dev) over [pgx/v5](https://github.com/jackc/pgx), [coreos/go-oidc](https://github.com/coreos/go-oidc) for JWT validation, [golang-migrate](https://github.com/golang-migrate/migrate) for schema (embedded). Errors are RFC 7807 problem+json. Logs via `slog` JSON.
- **Frontend:** Nuxt 3 SPA (`ssr: false`), Pinia, Tailwind, [oidc-client-ts](https://github.com/authts/oidc-client-ts). Workday-only schedule grid is hand-rolled with HTML5 pointer events — no heavy DnD library.
- **Auth:** browser does the OIDC PKCE dance against your IdP; backend validates the JWT on every `/api/*` request and upserts the user (and a corresponding `people` row) on each call.
- **One image:** the SPA is built with `nuxt generate`, embedded into the Go binary at compile time, and served from the same listener as the API.

## Development setup

The dev workflow mounts source into containers and reloads on change. You don't need Go or Node on the host.

```bash
git clone https://github.com/florianwenzel/levitate.git
cd levitate
cp .env.example .env
docker compose up -d --build
```

Services that come up:

| Service     | URL                                           | Notes                                                                            |
| ----------- | --------------------------------------------- | -------------------------------------------------------------------------------- |
| Frontend    | http://localhost:3000                         | Nuxt dev server with HMR                                                         |
| Backend     | http://localhost:8080/healthz                 | Go binary via `go run` against bind-mounted source                               |
| Postgres 16 | `postgres://levitate:levitate@localhost:5432/levitate` | Migrations applied automatically on backend boot                          |
| Keycloak    | http://localhost:8081 (admin / admin)         | The `levitate` realm is imported on first boot from `keycloak/realm-levitate.json` |

Two seeded users:

| User                 | Password | Role     |
| -------------------- | -------- | -------- |
| admin@example.com    | admin    | admin    |
| member@example.com   | member   | member   |

After editing `go.mod` or `package.json`, rebuild: `docker compose up -d --build`. Plain code changes hot-reload.

### Running the e2e suite

The e2e suite is Cypress; it spins up against the running compose stack.

```bash
cd e2e
npm install
npx cypress run         # headless
# or:
npx cypress open        # interactive
```

Tests reset the database between each spec via `POST /api/test/reset` (gated by `LEVITATE_ALLOW_TEST_RESET=true`, set in `docker-compose.yml`).

### Regenerating sqlc

```bash
docker run --rm -v "$(pwd)/backend:/src" -w /src sqlc/sqlc:latest generate
```

## Building the production image yourself

```bash
docker build -t levitate:local .
```

The multi-stage Dockerfile:
1. `nuxt generate`s the SPA in a Node stage.
2. Drops the SPA into the Go module's embed directory.
3. Compiles a static Go binary in a Go-Alpine stage (`-ldflags='-s -w'`, `CGO_ENABLED=0`).
4. Copies the binary into a slim Alpine runtime running as non-root.

Final image: ~30 MB, multi-arch (linux/amd64 + linux/arm64 in CI).

## Tech stack

- Go 1.25 · chi · sqlc · pgx/v5 · coreos/go-oidc · golang-migrate · slog
- Vue 3 (`<script setup>`) · Nuxt 3 SPA · Pinia · Tailwind · oidc-client-ts
- Postgres 16 · OIDC (any provider; reference: Keycloak 26)
- Cypress 13 (87 tests across 17 specs, ~35 s)

## Contributing

Issues and PRs welcome. The project is built in vertical slices — each PR should leave the app working end-to-end and the e2e suite green.

## License

[MIT](LICENSE) © Florian Wenzel.
