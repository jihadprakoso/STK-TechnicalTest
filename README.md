# Menu Tree

A menu-management application with a searchable hierarchical tree, drag-and-drop organization, a Go REST API, and PostgreSQL persistence.

## Technology

- **Frontend:** Next.js, React, TypeScript, Tailwind CSS utilities, and responsive component CSS.
- **Backend:** Go, Gin, GORM, and PostgreSQL, organized into handler, service, and repository layers.
- **API documentation:** Swagger UI at `/swagger/index.html`.
- **Database schema:** versioned SQL migrations in `backend/migrations`.

## Requirements

- Go 1.27 or later
- Node.js 20 or later and npm
- Docker with Docker Compose (recommended for PostgreSQL and full-stack development)

## Run with Docker Compose

Copy `.env.example` to `.env`, then start the development stack:

```sh
cp .env.example .env
docker compose up --build
```

The development setup starts PostgreSQL, applies migrations, starts the Go server with Air hot reload, and starts the Next.js development server.

- Frontend: <http://localhost:3000>
- API health: <http://localhost:8080/health>
- Swagger UI: <http://localhost:8080/swagger/index.html>

Source folders are mounted into the containers. PostgreSQL data is stored in the `pgdata` named volume.

To stop the stack while retaining database data:

```sh
docker compose down
```

To also remove local database data:

```sh
docker compose down --volumes
```

## Run locally

1. Start PostgreSQL:

   ```sh
   cp .env.example .env
   docker compose up -d db
   docker compose run --rm migrate
   ```

2. Configure the backend:

   ```sh
   cp backend/.env.example backend/.env
   ```

   Set `DATABASE_URL` to the PostgreSQL connection string for your local database. Then run:

   ```sh
   cd backend
   go run ./cmd/server
   ```

3. Configure and start the frontend in a separate terminal:

   ```sh
   cp frontend/.env.example frontend/.env.local
   cd frontend
   npm ci
   npm run dev
   ```

## Production with Docker Compose

1. Create `.env` from `.env.example`.
2. Set a strong, unique `POSTGRES_PASSWORD`.
3. Set `NEXT_PUBLIC_API_URL` to the public API URL that browsers can reach, including `/api`.
4. Start the production services:

   ```sh
   docker compose -f docker-compose.prod.yml up --build -d
   ```

The production Dockerfiles use build and runtime stages. The database is not published on a host port by the production Compose file. The frontend API URL is supplied at image build time.

## Environment variables

| Variable | Used by | Description |
|---|---|---|
| `POSTGRES_USER` | Compose/PostgreSQL | Database user |
| `POSTGRES_PASSWORD` | Compose/PostgreSQL | Database password; replace the example before production |
| `POSTGRES_DB` | Compose/PostgreSQL | Database name |
| `POSTGRES_PORT` | Development Compose | Host port for PostgreSQL |
| `BACKEND_PORT` | Production Compose | Host port for the API |
| `FRONTEND_PORT` | Production Compose | Host port for the web app |
| `DATABASE_URL` | Backend | PostgreSQL connection string |
| `PORT` | Backend | HTTP listen port (defaults to `8080`) |
| `NEXT_PUBLIC_API_URL` | Frontend | API base URL used by the browser |

Environment files containing actual credentials are ignored by Git. Do not commit secrets.

## API

All menu endpoints are under `/api`:

| Method | Path | Behavior |
|---|---|---|
| `GET` | `/menus` | Return all menus as a tree |
| `GET` | `/menus/{id}` | Get one menu |
| `POST` | `/menus` | Create a root or child menu |
| `PUT` | `/menus/{id}` | Update a menu name |
| `DELETE` | `/menus/{id}` | Delete a menu and its descendants |
| `PATCH` | `/menus/{id}/move` | Move a menu to another parent and position |
| `PATCH` | `/menus/{id}/reorder` | Reorder a menu within its current parent |
| `PUT` | `/menus/reorder` | Reorder all siblings (bulk compatibility endpoint) |

Names are trimmed, required, and limited to 100 Unicode characters. IDs and parent IDs are UUIDs. Move operations reject cycles and invalid positions.

## Database migrations

The development and production Compose configurations run migrations before starting the backend. To apply migrations manually against a configured PostgreSQL server, install the [golang-migrate CLI](https://github.com/golang-migrate/migrate) and run:

```sh
migrate -path backend/migrations -database "$DATABASE_URL" up
```

## Tests and checks

Run the backend unit and handler tests:

```sh
cd backend
go test ./...
go vet ./...
```

Repository integration tests use a dedicated PostgreSQL database configured with `TEST_DATABASE_URL`; the tests create a temporary `menus` table. Do not point this variable at a production database.

Run frontend checks:

```sh
cd frontend
npm ci
npm test
npm run lint
npm run build
```

## Project structure

```text
backend/
  cmd/server/       API entry point
  internal/         configuration, handlers, models, repositories, services
  migrations/       PostgreSQL schema migrations
  docs/             generated Swagger documentation
frontend/
  public/           UI assets
  src/app/          Next.js page and styles
docker-compose.yml  development stack
docker-compose.prod.yml
```

## Submission checklist

- Push the completed source to a public GitHub or GitLab repository.
- Verify no local environment files or secrets are included in the published history.
- Add screenshots or a short demo video to the repository.
- Share the public repository URL with the evaluator.
