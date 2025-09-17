package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"adm-backend/internal/ids"
	"adm-backend/internal/panbagnat"
	"adm-backend/internal/store"

	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	Sessions     *store.SessionStore
	Client       *panbagnat.Client
	ServiceToken string
}

type sessionResponse struct {
	ID             string    `json:"id"`
	Label          string    `json:"label"`
	StartAt        time.Time `json:"start_at"`
	EndAt          time.Time `json:"end_at"`
	Status         string    `json:"status"`
	IsOngoing      bool      `json:"is_ongoing"`
	StudentCount   int       `json:"student_count"`
	ValidatedCount int       `json:"validated_count"`
}

type listSessionsResponse struct {
	Sessions []sessionResponse `json:"sessions"`
}

type createSessionRequest struct {
	Label   string    `json:"label"`
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
}

type createSessionResponse struct {
	Session sessionResponse `json:"session"`
}

// RegisterAdminRoutes declares the admin-facing HTTP endpoints using a chi router.
func RegisterAdminRoutes(r chi.Router, handler *AdminHandler) {
	r.Get("/sessions", handler.handleListSessions)
	r.Post("/sessions", handler.handleCreateSession)
}

func (h *AdminHandler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	summaries, err := h.Sessions.ListSummaries(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	now := time.Now().UTC()
	resp := listSessionsResponse{Sessions: make([]sessionResponse, 0, len(summaries))}
	for _, summary := range summaries {
		resp.Sessions = append(resp.Sessions, toSessionResponse(summary, now))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *AdminHandler) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	if h.Client == nil {
		respondError(w, http.StatusInternalServerError, errors.New("pan bagnat client not configured"))
		return
	}

	if r.Body == nil {
		http.Error(w, "missing body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	payload.Label = strings.TrimSpace(payload.Label)
	if payload.Label == "" {
		payload.Label = defaultLabelFor(payload.StartAt)
	}

	if payload.StartAt.IsZero() || payload.EndAt.IsZero() {
		http.Error(w, "start_at and end_at are required", http.StatusBadRequest)
		return
	}
	if !payload.EndAt.After(payload.StartAt) {
		http.Error(w, "end_at must be after start_at", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	if payload.EndAt.Before(now) {
		http.Error(w, "end_at cannot be in the past", http.StatusBadRequest)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" && h.ServiceToken != "" {
		authHeader = h.ServiceToken
	}
	if authHeader == "" {
		http.Error(w, "missing Authorization header", http.StatusUnauthorized)
		return
	}
	students, err := h.Client.ListAllUsers(r.Context(), authHeader)
	if err != nil {
		respondError(w, http.StatusBadGateway, err)
		return
	}

	logins := dedupeLogins(students)

	sessionID, err := ids.New("adm_session")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	status := store.SessionStatus("draft")
	publishedAt := sql.NullTime{}
	if !payload.StartAt.After(now) {
		status = store.SessionStatus("active")
		publishedAt = sql.NullTime{Time: now, Valid: true}
	}

	createdBy := r.Header.Get("X-User-Login")
	if createdBy == "" {
		createdBy = "unknown_admin"
	}

	params := store.CreateSessionParams{
		ID:             sessionID,
		Label:          payload.Label,
		StartAt:        payload.StartAt,
		EndAt:          payload.EndAt,
		Status:         status,
		CreatedByLogin: createdBy,
		PublishedAt:    publishedAt,
	}

	if err := h.Sessions.InsertSessionWithStudents(r.Context(), params, logins); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	summaries, err := h.Sessions.ListSummaries(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	var created store.SessionSummary
	for _, summary := range summaries {
		if summary.ID == sessionID {
			created = summary
			break
		}
	}

	if created.ID == "" {
		respondError(w, http.StatusInternalServerError, sql.ErrNoRows)
		return
	}

	writeJSON(w, http.StatusCreated, createSessionResponse{Session: toSessionResponse(created, time.Now().UTC())})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, err error) {
	type errorResponse struct {
		Error string `json:"error"`
	}
	writeJSON(w, status, errorResponse{Error: err.Error()})
}

func toSessionResponse(summary store.SessionSummary, now time.Time) sessionResponse {
	isOngoing := (now.After(summary.StartAt) || now.Equal(summary.StartAt)) && (now.Before(summary.EndAt) || now.Equal(summary.EndAt))

	return sessionResponse{
		ID:             summary.ID,
		Label:          summary.Label,
		StartAt:        summary.StartAt,
		EndAt:          summary.EndAt,
		Status:         string(summary.Status),
		IsOngoing:      isOngoing,
		StudentCount:   summary.StudentCount,
		ValidatedCount: summary.ValidatedCount,
	}
}

func dedupeLogins(users []panbagnat.User) []string {
	seen := make(map[string]string)
	for _, user := range users {
		login := strings.TrimSpace(user.FtLogin)
		if login == "" {
			continue
		}
		seen[strings.ToLower(login)] = login
	}

	logins := make([]string, 0, len(seen))
	for _, login := range seen {
		logins = append(logins, login)
	}
	sort.Strings(logins)
	return logins
}

func defaultLabelFor(start time.Time) string {
	year := start.In(time.UTC).Year()
	if year <= 0 {
		year = time.Now().UTC().Year()
	}
	return "ADM " + strconv.Itoa(year)
}
