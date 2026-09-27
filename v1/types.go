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
// If an error occurs, it uses errorHandler, or when that is nil, the error handler of the
// router serving the request. Without either, it sends the ErrorResponse with its own
// status code, or 500 when that is not set. A nil response with no error sends 204 No Content.
func (rh RequestHandler[T]) ServeHTTPWithErrorHandler(errorHandler ErrorHandlerType[any]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, errResp := rh(r)

		if errResp != nil {
			sendError(w, r, errResp, errorHandler)
			return
		}

		if resp == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if resp.StatusCode == 0 {
			resp.StatusCode = http.StatusOK
		}
		sendResponse(w, resp)
	}
}

// sendError writes the response for errResp. It uses errorHandler when set, or else the
// router's error handler from the request context. It falls back to sending errResp
// itself when there is no error handler or the error handler returns nil. When the
// error handler's response has no status code, the ErrorResponse status code is used.
func sendError(w http.ResponseWriter, r *http.Request, errResp *ErrorResponse, errorHandler ErrorHandlerType[any]) {
	if errorHandler == nil {
		errorHandler = errorHandlerFromContext(r.Context())
	}

	if errorHandler != nil {
		if resp := errorHandler(r, errResp); resp != nil {
			if resp.StatusCode == 0 {
				resp.StatusCode = errorStatusCode(errResp)
			}
			sendResponse(w, resp)
			return
		}
	}

	sendResponse(w, &HttpResponse[*ErrorResponse]{
		StatusCode: errorStatusCode(errResp),
		Body:       errResp,
	})
}

// errorStatusCode returns the status code set on errResp, or 500 when it is not set.
func errorStatusCode(errResp *ErrorResponse) int {
	if errResp.StatusCode == 0 {
		return http.StatusInternalServerError
	}
	return errResp.StatusCode
}
