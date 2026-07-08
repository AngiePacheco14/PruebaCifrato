package dto

// ErrorResponse is the JSON body returned for any non-2xx response.
type ErrorResponse struct {
	Error string `json:"error"`
}
