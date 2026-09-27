package chirpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/go-chi/chi/v5"
	"github.com/iambpn/chirpc/internal/jsonbody"
	"github.com/iambpn/chirpc/internal/query"
	"github.com/iambpn/chirpc/internal/tags"
)

// NoBody is the Body type of a TypedHandler for requests without a body.
type NoBody struct{}

// NoQuery is the Query type of a TypedHandler for requests without query parameters.
type NoQuery struct{}

// Request holds a request whose body and query parameters are already decoded.
type Request[Body, Query any] struct {
	// Body is the decoded JSON request body. It is the zero value when Body is NoBody.
	Body Body
	// Query holds the decoded URL query parameters. It is the zero value when Query is NoQuery.
	Query Query
	// HTTP is the original request.
	HTTP *http.Request
}

// Param returns the URL path parameter with the given name, such as "id" for "/users/{id}".
func (r *Request[Body, Query]) Param(name string) string {
	return chi.URLParam(r.HTTP, name)
}

// Context returns the request's context.
func (r *Request[Body, Query]) Context() context.Context {
	return r.HTTP.Context()
}

// TypedHandler handles a request whose body and query parameters are already decoded.
// It returns a plain error. See AddTypedHandler for how errors are sent.
type TypedHandler[Body, Query, Res any] func(req *Request[Body, Query]) (*HttpResponse[Res], error)

// AddTypedHandler registers a handler whose request body and query types come from its
// type parameters, so the generated TypeScript always matches what the handler reads.
//
// Before the handler runs, the JSON body is decoded into Body and the URL query into Query.
// Use NoBody or NoQuery when the request has none. Both must be structs or pointers to
// structs. Query fields are read with the same names as the TypeScript members. A required
// field that is missing from the query or the body is reported, including fields of nested
// objects in the body. A field is required unless it has omitempty, omitzero, or
// tsOptional:"true", which is the same rule the generated TypeScript uses. A request that cannot be decoded gets a 400
// ErrorResponse, which goes through the router's error handler like any other error.
//
// When the handler returns an error, an *ErrorResponse in the error chain is sent as it
// is. Any other error becomes a 500 ErrorResponse with a general message, and the
// original error is kept in its Cause field for the error handler.
//
// It panics when the Body or Query type cannot be used.
func AddTypedHandler[Body, Query, Res any](r *RPCRouter, method HttpMethods, path string, handler TypedHandler[Body, Query, Res], middlewares ...MiddlewareType) {
	bodyType := reflect.TypeFor[Body]()
	queryType := reflect.TypeFor[Query]()
	hasBody := bodyType != reflect.TypeFor[NoBody]()
	hasQuery := queryType != reflect.TypeFor[NoQuery]()

	var bodyChecker *jsonbody.Checker
	if hasBody {
		if tags.Dereference(bodyType).Kind() != reflect.Struct {
			panic(fmt.Sprintf("chirpc: the body type must be a struct or a pointer to a struct, but got %s.", bodyType))
		}
		bodyChecker = jsonbody.NewChecker(bodyType)
	}

	var queryDecoder *query.Decoder
	if hasQuery {
		decoder, err := query.NewDecoder(tags.Dereference(queryType))
		if err != nil {
			panic(fmt.Sprintf("chirpc: the query type %s cannot be used: %s.", queryType, err))
		}
		queryDecoder = decoder
	}

	requestHandler := func(httpReq *http.Request) (*HttpResponse[Res], *ErrorResponse) {
		req := &Request[Body, Query]{HTTP: httpReq}

		if hasBody {
			if errResp := decodeBody(httpReq, bodyChecker, &req.Body); errResp != nil {
				return nil, errResp
			}
		}
		if hasQuery {
			if errResp := decodeQuery(queryDecoder, httpReq, reflect.ValueOf(&req.Query).Elem()); errResp != nil {
				return nil, errResp
			}
		}

		resp, err := handler(req)
		if err != nil {
			return nil, toErrorResponse(err)
		}
		return resp, nil
	}

	schema := AddHandler(r, method, path, RequestHandler[Res](requestHandler), middlewares...).Schema
	if hasBody {
		schema.SetBodyType(reflect.Zero(bodyType).Interface())
	}
	if hasQuery {
		schema.SetQueryType(reflect.Zero(queryType).Interface())
	}
}

// decodeBody decodes the JSON request body into target. It returns a 400 ErrorResponse
// when the body is missing or null, is not valid JSON for the target type, or leaves
// out a required field. checker finds the missing required fields.
func decodeBody(httpReq *http.Request, checker *jsonbody.Checker, target any) *ErrorResponse {
	data, err := io.ReadAll(httpReq.Body)
	if err != nil {
		return &ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Errors:     []string{"The request body could not be read."},
			Cause:      err,
		}
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return &ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Errors:     []string{"The request body is missing."},
		}
	}

	if err := json.Unmarshal(data, target); err != nil {
		return jsonErrorResponse(err)
	}

	if problems := checker.Check(data); problems != nil {
		return &ErrorResponse{
			StatusCode:       http.StatusBadRequest,
			Errors:           []string{"The request body is not valid."},
			ValidationErrors: problems,
		}
	}
	return nil
}

// jsonErrorResponse describes a JSON decoding error as a 400 ErrorResponse.
// A value of the wrong type is reported for its field.
func jsonErrorResponse(err error) *ErrorResponse {
	errResp := &ErrorResponse{StatusCode: http.StatusBadRequest, Cause: err}

	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &typeErr) && typeErr.Field != "":
		errResp.Errors = []string{"The request body is not valid."}
		errResp.ValidationErrors = map[string][]string{
			typeErr.Field: {fmt.Sprintf("This value must be %s.", jsonKindName(typeErr.Type))},
		}
	default:
		errResp.Errors = []string{"The request body is not valid JSON."}
	}
	return errResp
}

// decodeQuery decodes the URL query into target, which is the Query value of a Request.
// It allocates target first when Query is a pointer type.
func decodeQuery(decoder *query.Decoder, httpReq *http.Request, target reflect.Value) *ErrorResponse {
	if target.Kind() == reflect.Pointer {
		target.Set(reflect.New(target.Type().Elem()))
		target = target.Elem()
	}

	problems := decoder.Decode(httpReq.URL.Query(), target)
	if problems == nil {
		return nil
	}
	return &ErrorResponse{
		StatusCode:       http.StatusBadRequest,
		Errors:           []string{"The query parameters are not valid."},
		ValidationErrors: problems,
	}
}

// jsonKindName describes the JSON value that a Go type expects, for error messages.
func jsonKindName(typ reflect.Type) string {
	switch tags.Dereference(typ).Kind() {
	case reflect.Bool:
		return "a boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "a number"
	case reflect.String:
		return "a string"
	case reflect.Slice, reflect.Array:
		return "an array"
	default:
		return "an object"
	}
}
