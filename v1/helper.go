package chirpc

import (
	"encoding/json"
	"net/http"
	"reflect"
)

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
