package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"adm-backend/internal/ids"
)

type SessionStatus string

type SessionSummary struct {
	ID             string
	Label          string
	StartAt        time.Time
	EndAt          time.Time
	Status         SessionStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	StudentCount   int
	ValidatedCount int
}

type Session struct {
	ID          string
	Label       string
	StartAt     time.Time
	EndAt       time.Time
	Status      SessionStatus
	CreatedBy   string
	PublishedAt sql.NullTime
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateSessionParams struct {
	ID             string
	Label          string
	StartAt        time.Time
	EndAt          time.Time
	Status         SessionStatus
	CreatedByLogin string
	PublishedAt    sql.NullTime
}

type SessionStore struct {
	db *sql.DB
}

func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) ListSummaries(ctx context.Context) ([]SessionSummary, error) {
	const query = `
        SELECT
            s.id,
            s.label,
            s.start_at,
            s.end_at,
            s.status,
            s.created_at,
            s.updated_at,
            COALESCE(COUNT(ss.id), 0) AS student_count,
            COALESCE(SUM(CASE WHEN ss.status = 'validated' THEN 1 ELSE 0 END), 0) AS validated_count
        FROM adm_sessions s
        LEFT JOIN adm_student_sessions ss ON ss.adm_session_id = s.id
        GROUP BY s.id
        ORDER BY s.start_at DESC;
    `

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionSummary
	for rows.Next() {
		var summary SessionSummary
		if err := rows.Scan(
			&summary.ID,
			&summary.Label,
			&summary.StartAt,
			&summary.EndAt,
			&summary.Status,
			&summary.CreatedAt,
			&summary.UpdatedAt,
			&summary.StudentCount,
			&summary.ValidatedCount,
		); err != nil {
			return nil, fmt.Errorf("scan session summary: %w", err)
		}
		sessions = append(sessions, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session summaries: %w", err)
	}

	return sessions, nil
}

func (s *SessionStore) InsertSessionWithStudents(ctx context.Context, params CreateSessionParams, studentLogins []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	const insertSession = `
        INSERT INTO adm_sessions (
            id, label, start_at, end_at, status, configuration,
            created_by_login, published_at, created_at, updated_at
        ) VALUES ($1,$2,$3,$4,$5,NULL,$6,$7,NOW(),NOW());
    `

	if _, err := tx.ExecContext(
		ctx,
		insertSession,
		params.ID,
		params.Label,
		params.StartAt,
		params.EndAt,
		params.Status,
		params.CreatedByLogin,
		params.PublishedAt,
	); err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	if len(studentLogins) > 0 {
		const insertStudent = `
            INSERT INTO adm_student_sessions (
                id, adm_session_id, student_login, status, current_revision,
                locked_by_student, locked_by_admin, created_at, updated_at
            ) VALUES ($1,$2,$3,'not_started',1,false,false,NOW(),NOW());
        `

		stmt, err := tx.PrepareContext(ctx, insertStudent)
		if err != nil {
			return fmt.Errorf("prepare student insert: %w", err)
		}
		defer stmt.Close()

		for _, login := range studentLogins {
			if login == "" {
				continue
			}
			studentID, err := generateStudentSessionID()
			if err != nil {
				return fmt.Errorf("generate student session id: %w", err)
			}
			if _, err := stmt.ExecContext(ctx, studentID, params.ID, login); err != nil {
				return fmt.Errorf("insert student session for %s: %w", login, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit session creation: %w", err)
	}
	return nil
}

func generateStudentSessionID() (string, error) {
	return ids.New("adm_student_session")
}
