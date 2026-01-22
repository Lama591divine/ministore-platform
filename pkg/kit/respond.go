package kit

import (
	"encoding/json"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
)

type ErrorResponse struct {
	Error     string `json:"error"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, r *http.Request, status int, msg string, details any) {
	reqID := chimw.GetReqID(r.Context())
	WriteJSON(w, status, ErrorResponse{
		Error:     msg,
		Details:   details,
		RequestID: reqID,
	})
}
