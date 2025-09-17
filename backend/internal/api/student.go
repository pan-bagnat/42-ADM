package api

import "net/http"

// RegisterStudentRoutes attaches student-facing handlers to the provided mux.
func RegisterStudentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/student/sessions/current", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"session": null, "message": "Student session endpoint placeholder"}`))
	})

	mux.HandleFunc("/student/sessions/current/questionnaire", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		if _, err := w.Write([]byte(`{"message": "Questionnaire submission placeholder"}`)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/student/sessions/current/submit", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		if _, err := w.Write([]byte(`{"message": "Submit for validation placeholder"}`)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
