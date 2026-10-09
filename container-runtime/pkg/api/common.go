package api

// ErrorResponse is the standard error format for all API endpoints.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
