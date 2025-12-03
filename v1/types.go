package chirpc

import (
	"net/http"
)

// HttpResponse represents an HTTP response with a generic body type.
// StatusCode is the HTTP status code.
// Body contains the response payload.
// Headers is a map of response headers.
type HttpResponse[T any] struct {
	StatusCode int
	Body       T
	Headers    map[string]string
}

// ErrorResponse represents a structured error response with status code, error messages,
// and optional field-level validation errors.
type ErrorResponse struct {
	StatusCode       int                 `json:"statusCode,omitempty"`
	Errors           []string            `json:"errors,omitempty"`
	ValidationErrors map[string][]string `json:"validationErrors,omitempty"`
}

// MiddlewareType is a type alias for a middleware function that wraps an http.Handler.
type MiddlewareType = func(http.Handler) http.Handler

// ErrorHandlerType is a type alias for a function that handles errors and returns an HttpResponse.
type ErrorHandlerType[T any] = func(*http.Request, *ErrorResponse) *HttpResponse[T]

// RequestHandler defines a handler function that processes an HTTP request and returns an HttpResponse or error.
type RequestHandler[T any] func(*http.Request) (*HttpResponse[T], *ErrorResponse)

// ServeHTTPWithErrorHandler wraps the RequestHandler with error handling logic.
// If an error occurs, it uses the provided errorHandler to generate a response.
// If errorHandler is nil, it returns a 500 Internal Server Error.
func (rh RequestHandler[T]) ServeHTTPWithErrorHandler(errorHandler ErrorHandlerType[any]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, errResp := rh(r)

		if errResp != nil {
			if errorHandler != nil {
				resp := errorHandler(r, errResp)
				sendResponse(w, resp)
				return
			}

			// if error handler is not set, then return default Error response with status code 500
			defaultHttpResp := &HttpResponse[*ErrorResponse]{
				StatusCode: http.StatusInternalServerError,
				Body:       errResp,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			}
			sendResponse(w, defaultHttpResp)
			return
		}

		sendResponse(w, resp)
	}
}
