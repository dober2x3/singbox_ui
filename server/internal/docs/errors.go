package docs

// ErrorResponse is the standard error response returned by the API.
type ErrorResponse struct {
	Error   string `json:"error" example:"internal server error"`
	Message string `json:"message,omitempty" example:"detailed error description"`
}
