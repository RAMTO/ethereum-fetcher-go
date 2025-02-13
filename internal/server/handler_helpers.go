package server

import (
	"encoding/json"
	"net/http"
)

// ResponseHelper wraps common JSON response functionality
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// ErrorResponse represents a standard error response structure
type ErrorResponse struct {
	Error string `json:"error"`
}

// respondWithError is a helper to send error responses
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, ErrorResponse{Error: message})
}

// validateRequest is a helper to validate and decode JSON requests
func validateRequest(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(&dst); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return false
	}
	return true
}
