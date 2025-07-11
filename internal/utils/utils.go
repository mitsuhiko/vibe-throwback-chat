package utils

import (
	"encoding/json"
	"log"
	"net/http"
)

func APIError(w http.ResponseWriter, code int, message string, err error) {
	h := w.Header()
	h.Del("Content-Length")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Type", "application/json")
	w.WriteHeader(code)

	data := map[string]string{"error": message}
	if err != nil {
		data["detail"] = err.Error()
	}

	json.NewEncoder(w).Encode(data)
}

// Respond with an HTTP error and log the error
func InternalServerError(w http.ResponseWriter, err error) {
	log.Printf("Internal server error during request handling: %v", err)
	APIError(w, http.StatusInternalServerError, "Internal server error", nil)
}

// Respond with an HTTP error and log the error
func BadRequestError(w http.ResponseWriter, message string, err error) {
	log.Printf("Bad request (%s) during request handling: %v", message, err)
	APIError(w, http.StatusBadRequest, message, err)
}

// DecodeJSON decodes JSON from the request body into the target structure
// If decoding fails, it automatically writes a bad request error response
func DecodeJSON(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		BadRequestError(w, "Invalid request body", err)
		return false
	}
	return true
}

// SendJSON sends a JSON response with 200 status code and proper content type
func SendJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(data)
}

// SendJSONWithStatus sends a JSON response with custom status code and proper content type
func SendJSONWithStatus(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}
