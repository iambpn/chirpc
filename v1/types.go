package chirpc

import "net/http"

type HttpResponse[T any] struct {
	StatusCode int
	Body       T
	Headers    map[string]string
}

// type alias for middleware function
type MiddlewareType = func(http.Handler) http.Handler
type ErrorHandlerType[T any] = func(*http.Request, error) HttpResponse[T]

type RequestHandler[T any] func(*http.Request) (*HttpResponse[T], error)

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
