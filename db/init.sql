-- ADM module database bootstrap
-- IDs follow Pan-Bagnat conventions: prefixed ULIDs stored as TEXT (e.g. adm_session_01H...)

SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Enumerated types ---------------------------------------------------------

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'adm_session_status') THEN
        CREATE TYPE adm_session_status AS ENUM ('draft', 'active', 'closed');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'adm_student_session_status') THEN
        CREATE TYPE adm_student_session_status AS ENUM (
            'not_started',
            'waiting_for_documents',
            'waiting_for_validation',
            'validated',
            'invalidated'
        );
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'adm_document_submission_status') THEN
        CREATE TYPE adm_document_submission_status AS ENUM (
            'pending',
            'under_review',
            'valid',
            'invalid'
        );
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'adm_timeline_event_type') THEN
        CREATE TYPE adm_timeline_event_type AS ENUM (
            'questionnaire_started',
            'questionnaire_completed',
            'files_submitted',
            'admin_review_started',
            'document_validated',
            'document_invalidated',
            'review_replied',
            'session_validated',
            'session_invalidated',
            'deadline_expired',
            'document_deleted',
            'generated_document_created'
        );
    END IF;
END$$;

-- Core tables --------------------------------------------------------------

CREATE TABLE IF NOT EXISTS adm_sessions (
    id                  TEXT PRIMARY KEY,
    label               TEXT NOT NULL,
    start_at            TIMESTAMPTZ NOT NULL,
    end_at              TIMESTAMPTZ NOT NULL,
    status              adm_session_status NOT NULL DEFAULT 'draft',
    configuration       JSONB,
    created_by_login    TEXT NOT NULL,
    published_at        TIMESTAMPTZ,
    closed_at           TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT adm_sessions_start_end_ck CHECK (end_at > start_at),
    CONSTRAINT adm_sessions_id_prefix CHECK (id LIKE 'adm_session_%')
);

CREATE UNIQUE INDEX IF NOT EXISTS adm_sessions_label_uniq
    ON adm_sessions (label);

CREATE TABLE IF NOT EXISTS adm_categories (
    id                  TEXT PRIMARY KEY,
    adm_session_id      TEXT NOT NULL REFERENCES adm_sessions(id) ON DELETE CASCADE,
    code                TEXT NOT NULL,
    label               TEXT NOT NULL,
    description         TEXT,
    questionnaire_logic JSONB,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT adm_categories_code_session_uniq UNIQUE (adm_session_id, code),
    CONSTRAINT adm_categories_id_prefix CHECK (id LIKE 'adm_category_%')
);

CREATE TABLE IF NOT EXISTS adm_document_requirements (
    id                  TEXT PRIMARY KEY,
    adm_session_id      TEXT NOT NULL REFERENCES adm_sessions(id) ON DELETE CASCADE,
    code                TEXT NOT NULL,
    title               TEXT NOT NULL,
    description         TEXT,
    accepted_mime_types TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    max_file_size_bytes BIGINT,
    reminder_order      SMALLINT,
    is_mandatory        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT adm_document_requirements_code_session_uniq UNIQUE (adm_session_id, code),
    CONSTRAINT adm_document_requirements_id_prefix CHECK (id LIKE 'adm_document_requirement_%')
);

-- Many-to-many link between categories and requirements to avoid duplication.
CREATE TABLE IF NOT EXISTS adm_category_requirements (
    category_id             TEXT NOT NULL REFERENCES adm_categories(id) ON DELETE CASCADE,
    document_requirement_id TEXT NOT NULL REFERENCES adm_document_requirements(id) ON DELETE CASCADE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (category_id, document_requirement_id)
);

CREATE TABLE IF NOT EXISTS adm_student_sessions (
    id                      TEXT PRIMARY KEY,
    adm_session_id          TEXT NOT NULL REFERENCES adm_sessions(id) ON DELETE CASCADE,
    student_login           TEXT NOT NULL,
    category_id             TEXT REFERENCES adm_categories(id) ON DELETE SET NULL,
    status                  adm_student_session_status NOT NULL DEFAULT 'not_started',
    current_revision        INTEGER NOT NULL DEFAULT 1,
    locked_by_student       BOOLEAN NOT NULL DEFAULT FALSE,
    locked_by_admin         BOOLEAN NOT NULL DEFAULT FALSE,
    last_questionnaire_at   TIMESTAMPTZ,
    last_submitted_at       TIMESTAMPTZ,
    last_reviewed_at        TIMESTAMPTZ,
    invalidation_reason     TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT adm_student_sessions_revision_ck CHECK (current_revision > 0),
    CONSTRAINT adm_student_sessions_session_login_uniq UNIQUE (adm_session_id, student_login),
    CONSTRAINT adm_student_sessions_id_prefix CHECK (id LIKE 'adm_student_session_%')
);

CREATE INDEX IF NOT EXISTS adm_student_sessions_status_idx
    ON adm_student_sessions (status);

CREATE INDEX IF NOT EXISTS adm_student_sessions_login_idx
    ON adm_student_sessions (student_login);

CREATE TABLE IF NOT EXISTS adm_questionnaire_responses (
    id                  TEXT PRIMARY KEY,
    student_session_id  TEXT NOT NULL REFERENCES adm_student_sessions(id) ON DELETE CASCADE,
    revision_number     INTEGER NOT NULL,
    answers             JSONB NOT NULL,
    calculated_category TEXT REFERENCES adm_categories(id) ON DELETE SET NULL,
    submitted_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT adm_questionnaire_responses_revision_ck CHECK (revision_number > 0),
    CONSTRAINT adm_questionnaire_responses_unique UNIQUE (student_session_id, revision_number),
    CONSTRAINT adm_questionnaire_responses_id_prefix CHECK (id LIKE 'adm_questionnaire_response_%')
);

CREATE TABLE IF NOT EXISTS adm_document_submissions (
    id                      TEXT PRIMARY KEY,
    student_session_id      TEXT NOT NULL REFERENCES adm_student_sessions(id) ON DELETE CASCADE,
    document_requirement_id TEXT NOT NULL REFERENCES adm_document_requirements(id) ON DELETE CASCADE,
    revision_number         INTEGER NOT NULL,
    status                  adm_document_submission_status NOT NULL DEFAULT 'pending',
    storage_key             TEXT NOT NULL,
    file_name               TEXT NOT NULL,
    file_size_bytes         BIGINT,
    checksum_sha256         TEXT,
    uploaded_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_by_login       TEXT NOT NULL,
    decision_by_login       TEXT,
    decision_at             TIMESTAMPTZ,
    admin_comment           TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT adm_document_submissions_revision_ck CHECK (revision_number > 0),
    CONSTRAINT adm_document_submissions_unique UNIQUE (student_session_id, document_requirement_id, revision_number),
    CONSTRAINT adm_document_submissions_id_prefix CHECK (id LIKE 'adm_document_submission_%')
);

CREATE INDEX IF NOT EXISTS adm_document_submissions_status_idx
    ON adm_document_submissions (status);

CREATE INDEX IF NOT EXISTS adm_document_submissions_requirement_idx
    ON adm_document_submissions (document_requirement_id);

CREATE TABLE IF NOT EXISTS adm_generated_documents (
    id                  TEXT PRIMARY KEY,
    student_session_id  TEXT NOT NULL REFERENCES adm_student_sessions(id) ON DELETE CASCADE,
    document_type       TEXT NOT NULL,
    storage_key         TEXT NOT NULL,
    file_name           TEXT NOT NULL,
    generated_by_login  TEXT NOT NULL,
    generated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT adm_generated_documents_unique UNIQUE (student_session_id, document_type),
    CONSTRAINT adm_generated_documents_id_prefix CHECK (id LIKE 'adm_generated_document_%')
);

CREATE TABLE IF NOT EXISTS adm_timeline_events (
    id                  TEXT PRIMARY KEY,
    student_session_id  TEXT NOT NULL REFERENCES adm_student_sessions(id) ON DELETE CASCADE,
    event_type          adm_timeline_event_type NOT NULL,
    payload             JSONB,
    created_by_login    TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT adm_timeline_events_id_prefix CHECK (id LIKE 'adm_timeline_event_%')
);

CREATE INDEX IF NOT EXISTS adm_timeline_events_student_idx
    ON adm_timeline_events (student_session_id, created_at DESC);

CREATE TABLE IF NOT EXISTS adm_storage_cleanup_queue (
    id               BIGSERIAL PRIMARY KEY,
    storage_key      TEXT NOT NULL,
    scheduled_for    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    enqueued_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at     TIMESTAMPTZ,
    failure_reason   TEXT
);

-- Automatic timestamp maintenance -----------------------------------------

CREATE OR REPLACE FUNCTION adm_touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach the trigger to tables that expose updated_at.
DO $$
DECLARE
    tbl TEXT;
BEGIN
    FOREACH tbl IN ARRAY ARRAY[
        'adm_sessions',
        'adm_categories',
        'adm_document_requirements',
        'adm_student_sessions',
        'adm_document_submissions'
    ]
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_trigger WHERE tgname = tbl || '_touch_updated_at'
        ) THEN
            EXECUTE format(
                'CREATE TRIGGER %I_touch_updated_at BEFORE UPDATE ON %I
                 FOR EACH ROW EXECUTE FUNCTION adm_touch_updated_at()',
                tbl,
                tbl
            );
        END IF;
    END LOOP;
END;
$$;
