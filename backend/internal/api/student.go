package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterStudentRoutes attaches student-facing handlers to the provided chi router.
func RegisterStudentRoutes(r chi.Router) {
	r.Get("/sessions/current", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"session": null, "message": "Student session endpoint placeholder"}`))
	})

	r.Post("/sessions/current/questionnaire", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		if _, err := w.Write([]byte(`{"message": "Questionnaire submission placeholder"}`)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.Post("/sessions/current/submit", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		if _, err := w.Write([]byte(`{"message": "Submit for validation placeholder"}`)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
