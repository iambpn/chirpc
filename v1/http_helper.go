// Package chirpc provides helper utilities for handling HTTP responses,
// including writing typed JSON payloads, setting headers, and basic error handling.
package chirpc

import (
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"reflect"
)

/*
Helper functions for handling HTTP responses.
*/

// sendResponse writes the provided HttpResponse to the ResponseWriter by setting
// headers, marshalling the body to JSON when possible, and writing the status code.
// If JSON marshalling fails or the body is not marshallable, it writes an HTTP 500 error.
func sendResponse[T any](w http.ResponseWriter, resp *HttpResponse[T]) {
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}

	tType := reflect.TypeOf(resp.Body)

	if tType.Kind() == reflect.Pointer {
		tType = tType.Elem()
	}

	if isJSONMarshable(tType.Kind()) {
		outBytes, err := json.Marshal(resp.Body)
		if err != nil {
			http.Error(w, "an error occurred while marshalling payload", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(resp.StatusCode)
		w.Write(outBytes)
		return
	}

	http.Error(w, "payload is not marshallable", http.StatusInternalServerError)
}

// sendStreamBytes writes byte chunks from a stream using HTTP chunked transfer.
// It flushes each chunk when possible and stops when the stream closes or request context is done.
func sendStreamBytes(w http.ResponseWriter, r *http.Request, resp *StreamResponse) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not supported by response writer", http.StatusInternalServerError)
		return
	}

	headers := map[string]string{
		"Content-Type":      "application/x-ndjson",
		"Cache-Control":     "no-cache",
		"Connection":        "keep-alive",
		"Transfer-Encoding": "chunked",
	}

	maps.Copy(headers, resp.Headers)

	for k, v := range headers {
		w.Header().Set(k, v)
	}

	statusCode := resp.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	w.WriteHeader(statusCode)

	for {
		select {
		case <-r.Context().Done():
			return
		case chunk, ok := <-resp.Stream:
			if !ok {
				return
			}

			_, err := w.Write(chunk)
			if err != nil {
				return
			}

			flusher.Flush()
		}
	}
}

// ReaderToStream reads from r in chunks and sends each chunk to the returned
// channel. cleanup is called (via defer) once the reader is exhausted or returns
// an error, making it suitable for releasing resources such as open files (e.g. file.Close).
// cleanup may be nil. bufSize optionally sets the read buffer size in bytes (default: 8 KiB).
// The channel is closed automatically when reading finishes.
func ReaderToStream(r io.Reader, cleanup func() error, bufSize ...int) chan []byte {
	stream := make(chan []byte)

	chunkSize := 8 * 1024
	if len(bufSize) > 0 && bufSize[0] > 0 {
		chunkSize = bufSize[0]
	}

	go func() {
		defer close(stream)
		if cleanup != nil {
			defer cleanup()
		}

		buffer := make([]byte, chunkSize)

		for {
			n, err := r.Read(buffer)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buffer[:n])
				stream <- chunk
			}

			if err != nil {
				return
			}
		}
	}()

	return stream
}

// isJSONMarshable reports whether the given reflect.Kind can be marshaled to JSON
// by the encoding/json package. This checks the Kind itself, not dereferenced types.
func isJSONMarshable(v reflect.Kind) bool {
	// Check for basic types that can be marshaled to JSON
	switch v {
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
		return true
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool:
		return true
	case reflect.Interface, reflect.Pointer:
		return true
	default:
		return false
	}
}
