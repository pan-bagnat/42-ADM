# ADM Module Architecture

## Overview
The ADM module handles yearly administrative paperwork between students and school staff. It provides:
- A **backend API** that exposes separate surfaces for students and administrators.
- Two **frontend applications** (student and admin) backed by the API.
- A **persistent database** to track sessions, document requirements, submissions, and event history.
- Containerized deployment through docker-compose to run the four services (database, backend, admin UI, student UI).

The module receives annual configuration from administrators and coordinates a "ping-pong" review process until all required documents are validated. Once validation completes, generated documents become available to the student, and raw uploads are deleted.

## Actors & Roles
- **Student**: authenticated user submitting paperwork within an ADM session.
- **Administrator**: school staff onboarding sessions, reviewing submissions, validating/invalidation documents.
- **System jobs**: background workers enforcing session deadlines, cleaning up files, and producing generated attestations.

Authentication and user directory (login, names, etc.) are provided by the parent Pan-Bagnat orchestrator. The ADM backend trusts incoming JWTs/access tokens and stores foreign user identifiers.

## Domain Concepts
### ADM Session (Yearly)
Represents a school-year window during which administrative documents are collected.
- `id`
- `label` (e.g. "ADM 2025")
- `start_at`, `end_at`
- `status`: draft | active | closed
- `configuration_version`
- `created_by_admin_id`

### Student Session
One per student per ADM session. Created automatically when the ADM session transitions to `active`.
- `id`
- `adm_session_id`
- `student_login`
- `category_id`
- `status`: not_started | waiting_for_documents | waiting_for_validation | validated | invalidated
- `locked_by_student`: bool (true when student has sent files and is awaiting admin action)
- `locked_by_admin`: bool (true when admin has finished review and needs student action)
- `current_revision`
- `last_submitted_at`
- `last_reviewed_at`
- `invalidation_reason` (only when status=invalidated)

### Category
Derived from student questionnaire. Controls which documents are required.
- `id`
- `adm_session_id`
- `label`
- `questionnaire_logic` (serialized JSON / rule engine source)
- `document_requirement_ids`

Questionnaire responses are stored per student session revision for auditability.

### Document Requirement
Defines one slot the student must fill.
- `id`
- `adm_session_id`
- `category_id` (nullable if shared across many categories)
- `code` (e.g. `assurance_maladie`)
- `title`
- `description`
- `accepted_mime_types`
- `max_file_size`
- `reminder_order`

### Document Submission
Represents the file the student uploaded for a requirement within a specific revision.
- `id`
- `student_session_id`
- `document_requirement_id`
- `revision_number`
- `storage_key`
- `file_name`
- `uploaded_at`
- `uploaded_by`
- `status`: pending | under_review | valid | invalid
- `admin_comment`
- `decision_by`
- `decision_at`

Submissions with status `valid` or `invalid` are immutable. When a session is invalidated, all requirements move back to `pending` with a new revision number and fresh upload URLs.

### Generated Document
Output issued by the school after validation.
- `id`
- `student_session_id`
- `document_type`: `attestation_inscription`, `certificat_scolarite`, ...
- `storage_key`
- `file_name`
- `generated_at`
- `generated_by`

### Timeline Event
Immutable audit trail of everything that happens.
- `id`
- `student_session_id`
- `event_type`: questionnaire_started, questionnaire_completed, files_submitted, admin_review_started, document_validated, document_invalidated, review_replied, session_validated, session_invalidated, deadline_expired, document_deleted, generated_document_created
- `payload` (JSON snapshot)
- `created_at`
- `created_by`

### Questionnaire Response
Stores answers per revision.
- `id`
- `student_session_id`
- `revision_number`
- `answers` (JSON)
- `calculated_category_id`
- `submitted_at`

## State Machines
### Student Session State Transitions
```
not_started --student completes questionnaire--> waiting_for_documents
waiting_for_documents --student submits all required docs + lock--> waiting_for_validation
waiting_for_validation --admin validates all docs--> validated
waiting_for_validation --admin invalidates at least one doc--> invalidated
invalidated --student uploads new documents--> waiting_for_validation
validated --admin reopens session--> waiting_for_documents (new revision)
```
Sessions auto-transition to `invalidated` with reason "session expired" when ADM session end date passes and status is neither validated nor invalidated.

### Document Submission State Transitions
```
pending --student uploads--> under_review (and student lock on requirement)
under_review --admin marks valid--> valid (immutable)
under_review --admin marks invalid--> invalid (mutable only after session invalidated)
valid --admin reopens session--> pending (new revision)
invalid --student resubmits after session invalidated--> pending (new revision)
```

## Flows
### ADM Session Lifecycle
1. Admin creates ADM session as draft, configures questionnaire, categories, document slots.
2. Admin publishes session (status `active`) with start/end dates.
3. Background job (or admin-triggered creation) creates `StudentSession` entries for all eligible students when session starts by querying Pan-Bagnat `/api/v1/admin/users`.
4. On end date, background job marks unfinished student sessions invalid with reason "The ADM session ended without validation".

### Student Flow
1. Student visits student front. If status `not_started`, they must complete the questionnaire.
2. Backend calculates category and required documents.
3. Student uploads files for each requirement. Each upload locks the slot; they can remove/replace only before hitting "Submit for review".
4. When all mandatory slots have files, the student locks the session (`waiting_for_validation`). Upload URLs become read-only.
5. Student waits for admin decision. They cannot edit anything until admin answers.
6. If validated, student sees generated documents. If invalidated, they see per-document reasons and unlocked invalid slots to resubmit.

### Admin Flow
1. Admin searches for a student session via login or filters by status/category.
2. Admin inspects submitted documents. For each requirement they choose valid/invalid; invalid selection forces a reason.
3. When all requirements are decided, admin clicks "Send answer". This updates session status and notifies the student.
4. Admin can later reopen a validated session, triggering new revision and requiring fresh uploads.

## Notifications & Real-time
- Email + in-app notifications for student when admin returns a decision.
- Optional admin digest for sessions waiting for validation.
- WebSocket/SSE channel for UI to refresh decisions in real time.

## Storage Strategy
- Use object storage (S3-compatible or MinIO in development) for binary files. Database stores only metadata (`storage_key`, checksums).
- Files uploaded by students are soft-deleted after validation and physically deleted by background job to comply with requirement.
- Generated documents stored in dedicated bucket/prefix and retained while session validated.

## API Surfaces
### Student API
- `GET /student/sessions/current` – fetch current session, status, questionnaire state, required docs.
- `POST /student/sessions/current/questionnaire` – submit questionnaire answers and lock in category.
- `POST /student/sessions/current/documents/:requirementId` – upload document (pre-signed URL workflow recommended).
- `POST /student/sessions/current/submit` – lock session and request validation.
- `POST /student/sessions/current/unlock` – allow edits before admin review (only before submit or if admin reopened).
- `GET /student/sessions/current/history` – timeline events.
- `GET /student/sessions/current/generated-documents` – download generated certificates once available.

### Admin API
- `GET /admin/sessions` – list ADM sessions.
- `POST /admin/sessions` – create session.
- `PATCH /admin/sessions/:id` – update schedule/config, publish/close.
- `POST /admin/sessions/:id/rebuild-student-sessions` – optional repair job.
- `GET /admin/student-sessions` – search by filters (login, status, category, etc.).
- `GET /admin/student-sessions/:id` – detailed view.
- `POST /admin/student-sessions/:id/review` – submit decisions per document requirement with reasons.
- `POST /admin/student-sessions/:id/reopen` – reopen a validated session (new revision).
- `POST /admin/student-sessions/:id/generate-documents` – trigger generation pipeline.

### Internal/Background API
- `POST /internal/jobs/process-session-expirations`
- `POST /internal/jobs/cleanup-storage`

## Permissions & Security
- Backend enforces role-based access using JWT claims (`role = student|admin`), scoping data to the caller.
- All storage operations go through backend-signed URLs to avoid direct bucket credentials on frontend.
- Backend validates file types, size limits, and ensures admin reviews include reasons for invalidation.

## Technology Stack (recommended)
- **Backend**: Go 1.22 service (chi or net/http) with a layered structure (`cmd/server`, `internal/api`, `internal/store`) and SQL migrations for PostgreSQL access (sqlc or GORM optional).
- **Database**: PostgreSQL 15.
- **Storage**: MinIO locally, S3 in production.
- **Student UI**: React + Vite (JavaScript, `.jsx`) with component library (Material UI / Chakra) for accessible forms.
- **Admin UI**: React + Vite (JavaScript) focusing on data tables, review workflow; share component library with student UI where possible.
- **Infrastructure**: Docker Compose for local dev. CI pipeline to build and test containers.

## Background Jobs
- **Session activation**: at start date, create `StudentSession` rows for all active students.
- **Session expiry**: nightly job to invalidate overdue sessions.
- **Storage cleanup**: delete raw uploads when session becomes validated; ensure generated docs remain available.
- **Notification dispatcher**: send emails/notifications on status changes.

## Open Questions
- Student population source: pulled from central directory, or dynamic when first student logs in?
- Generated documents: produced manually by admins and uploaded, or auto-generated by backend templates?
- Questionnaire complexity: do we need a rule builder UI or simple branching logic?
- Do we need to version document requirements mid-session (e.g., new doc required mid-year)? If yes, plan for re-sync of student sessions.
- Localization: do we handle multi-language UIs and notifications now?

## Next Steps
1. Finalize decisions on questionnaire engine and generated document process.
2. Implement database migrations and backend modules per schema above.
3. Scaffold frontend apps and integrate with backend APIs.
4. Wire notification channels and background jobs.
5. Configure CI/CD and secrets management for storage provider.
