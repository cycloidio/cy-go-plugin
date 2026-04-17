# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

A Cycloid platform plugin — a small HTTP server that serves fixture/demo data from a SQLite database. The Cycloid platform calls well-known endpoints (`/_cy/*`) and uses `widgets.yaml` to render dashboards backed by SQL queries against the plugin's SQLite DB.

The plugin is distributed as a Docker image and declared via `manifest.yaml`.

## Build & run

```bash
# Build binary
go build -o cy-go-plugin .

# Run locally (defaults: in-memory SQLite, port 8080)
./cy-go-plugin

# Run with a persistent DB file
DB_FILE=/tmp/plugin.db PORT=9090 ./cy-go-plugin

# Build and push Docker image
make docker-release        # pushes to cycloid/cy-go-plugin:VERSION
make docker-local          # pushes to localhost:5000/cycloid/cy-go-plugin:VERSION
```

There is no test suite in this repository.

## Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `DB_FILE` | *(none — uses in-memory SQLite)* | Path to the SQLite DB file. The platform sets this to the mapped/mounted DB path. |
| `PORT` | `8080` | HTTP listen port. |

`DB_FILE` must match what the Cycloid platform mounts — the platform reads the SQLite file directly via the CLI. When `DB_FILE` is not set the server uses an in-memory database (useful for local dev/testing).

## Core HTTP API

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/_cy/ping` | Health check |
| POST | `/_cy/events` | Event hook (no-op) |
| DELETE | `/_cy/plugin` | Clears all data |
| POST | `/_cy/resync` | Clears then re-seeds all data |
| GET | `/sentry/iframe` | HTML iframe widget for Sentry preview |

## Code structure

Everything lives in `main.go`:

- **`main()`** — opens DB, runs `schema.sql` (embedded via `//go:embed`), seeds, registers routes
- **`seed()`** — inserts fixture organizations/projects/issues if the DB is empty (idempotent)
- **`clearData()`** — deletes all rows in dependency order (issues → projects → organizations)
- **`iframeHandler()`** — serves a static HTML table of the fixture Sentry issues

`schema.sql` is the source of truth for the DB schema and is embedded into the binary at build time.

## Dockerfile

Two-stage build: `golang:1.25.0` builder → `alpine:3.21` runtime. `CGO_ENABLED=0` is set because `modernc.org/sqlite` is a pure-Go SQLite implementation (no CGO required). The final image includes the binary, `*.yaml`, and `*.sql` files under `/plugin/`.

## widgets.yaml

Declares the dashboard widgets the Cycloid platform renders. Widgets execute SQL queries directly against the plugin's SQLite DB (`query:` field) or reference a plugin HTTP path (`query: "/sentry/iframe"` for iframes). Column names in queries must match the `value:` keys in the `columns:` list.
