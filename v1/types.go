package chirpc

import "net/http"

// HttpResponse represents an HTTP response with a generic body type.
// StatusCode is the HTTP status code.
// Body contains the response payload.
// Headers is a map of response headers.
type HttpResponse[T any] struct {
	StatusCode int
	Body       T
	Headers    map[string]string
}

// MiddlewareType is a type alias for a middleware function that wraps an http.Handler.
type MiddlewareType = func(http.Handler) http.Handler

// ErrorHandlerType is a type alias for a function that handles errors and returns an HttpResponse.
type ErrorHandlerType[T any] = func(*http.Request, error) HttpResponse[T]

// RequestHandler defines a handler function that processes an HTTP request and returns an HttpResponse or error.
type RequestHandler[T any] func(*http.Request) (*HttpResponse[T], error)

// ServeHTTPWithErrorHandler wraps the RequestHandler with error handling logic.
// If an error occurs, it uses the provided errorHandler to generate a response.
// If errorHandler is nil, it returns a 500 Internal Server Error.
func (rh RequestHandler[T]) ServeHTTPWithErrorHandler(errorHandler ErrorHandlerType[any]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := rh(r)

		if err != nil {
			if errorHandler != nil {
				resp := errorHandler(r, err)
				sendResponse(w, &resp)
				return
			}

			// if error handler is not set, return 500
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sendResponse(w, resp)
	}
}
