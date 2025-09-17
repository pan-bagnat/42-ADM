# Pan-Bagnat ADM Module

This repository hosts the administrative paperwork module for Pan-Bagnat. It coordinates the yearly submission, review, and validation workflow between students and school administrators.

## Repository Layout
- `backend/` – Go HTTP API exposing student and admin surfaces.
- `frontend-student/` – Student portal built with React (.jsx) + Vite.
- `frontend-admin/` – Admin dashboard skeleton built with React (.jsx) + Vite.
- `db/` – Database bootstrap artifacts.
- `docs/` – Architecture and design notes.
- `docker-compose.yml` – Orchestrates the four required services: database, backend, admin UI, student UI.

Refer to `docs/architecture.md` for detailed domain discussions, state machines, and open questions.

## Getting Started
1. Ensure Docker and Docker Compose are installed.
2. Build and start the stack:
   ```sh
   docker compose up --build
   ```
3. Services:
   - Backend API – `http://localhost:3000`
   - Admin UI – `http://localhost:8080`
   - Student UI – `http://localhost:8081`
   - PostgreSQL – `localhost:5432` (user/password: `adm`)

The frontends are compiled as static bundles served by Nginx. They communicate with the backend through the URL baked at build time (`VITE_BACKEND_URL`).

### Common Environment Variables
Backends and builds read the following variables:

| Variable | Service | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | backend | Connection string for PostgreSQL (example for local compose: `postgres://adm:adm@db:5432/adm?sslmode=disable`) |
| `PORT` | backend | HTTP port (defaults to 3000) |
| `CORS_ORIGIN` | backend | Comma-separated origins allowed to call the API (defaults to `http://localhost:8080,http://localhost:8081`) |
| `PAN_BAGNAT_API_BASE_URL` | backend | Base URL of the core Pan-Bagnat API used to fetch students |
| `PAN_BAGNAT_SERVICE_TOKEN` | backend | Optional Authorization header (e.g. `Bearer …`) used when the frontend does not supply one |
| `VITE_BACKEND_URL` | admin/student front builds | Base URL baked into the frontend bundles (defaults to deriving `http(s)://<host>:3000` in the browser) |

Override these via `.env` files or compose overrides as needed.

## Data Model Snapshot
Key aggregates are covered in detail in `docs/architecture.md`. Primary tables will include:
- `adm_sessions`
- `adm_categories`
- `adm_document_requirements`
- `adm_student_sessions`
- `adm_document_submissions`
- `adm_generated_documents`
- `adm_timeline_events`
- `adm_questionnaire_responses`

Each student session holds a status (`not_started`, `waiting_for_documents`, `waiting_for_validation`, `validated`, `invalidated`) and tracks per-document decisions with audit history. Admin decisions drive the ping-pong workflow until every requirement is validated.

## Development Roadmap
1. **Authentication integration** – hook into Pan-Bagnat SSO/JWT and propagate student/admin identities.
2. **Database migrations** – implement schema with a migration tool (e.g., Goose, Atlas, or sqlc-generated queries).
3. **Storage adapter** – abstract local and cloud object storage for document uploads/deletions.
4. **API implementation** – extend student/admin endpoints (questionnaire logic, document handling, validation workflows).
5. **Frontend features** – complete questionnaire UX, upload experiences, review screens, and notification surfaces.
6. **Notification/Job workers** – schedule expirations, send reminders, and handle session closure.
7. **Testing & CI** – add unit/integration tests and container builds to the pipeline.

## Contributing
- Follow idiomatic Go and React patterns for new code.
- Prefer schema-driven validation in the API layer (e.g., `go-playground/validator`, `ent`, custom validation).
- Keep Pan-Bagnat integration points (IDs, API contracts) in sync with the main orchestrator repository.
- Document new workflows in `docs/` as the module evolves.
