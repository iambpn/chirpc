package rpc

import (
	"fmt"
	"reflect"
)

// errorHandlerMethod is the ApiSchema key that holds the error response type.
const errorHandlerMethod = "ERROR_HANDLER"

// RouterRpcSchemas manages the handler schemas of one router and
// generates TypeScript types for registered RPC endpoints.
// Handler URLs are stored relative to the router. Full URLs are built
// from the mounted child routers when the TypeScript is generated.
type RouterRpcSchemas struct {
	schemas []*HandlerSchema
	mounts  []mountedSchemas

	// errorHandler is the error handler registered on this router, or nil.
	errorHandler *HandlerSchema
	// defaultErrorHandler describes the error body sent when no error handler is
	// registered. It is used only when this router is the root of the schema.
	defaultErrorHandler *HandlerSchema

	// typeOverrides maps Go types to the TypeScript type written for them.
	typeOverrides map[reflect.Type]string
}

// mountedSchemas is a child router's schema collection and the path it is mounted at.
type mountedSchemas struct {
	prefix  string
	schemas *RouterRpcSchemas
}

// Route is a registered handler resolved to its full URL. It holds the Go types
// that the TypeScript generator converts.
type Route struct {
	Method string
	// URL is the full chi URL pattern, such as "/users/{id}".
	URL    string
	Params []string
	// Response is the handler's HttpResponse struct type. Its Body field holds the response body type.
	Response reflect.Type
	Body     reflect.Type
	Query    reflect.Type
}

// RegisterHandler registers an RPC schema for type generation with its method, URL, and return type.
// It returns a HandlerSchema for optional body, query, and params enrichment.
func (r *RouterRpcSchemas) RegisterHandler(method, url string, fnVal any) (*HandlerSchema, error) {
	schema, err := BuildGoToTsSchema(method, url, fnVal)
	if err != nil {
		return nil, err
	}

	r.schemas = append(r.schemas, schema)
	return schema, nil
}

// RegisterErrorHandler records the error handler's response type for this router.
// A later call replaces the earlier one.
func (r *RouterRpcSchemas) RegisterErrorHandler(fnVal any) error {
	schema, err := BuildGoToTsSchema(errorHandlerMethod, "/", fnVal)
	if err != nil {
		return err
	}

	r.errorHandler = schema
	return nil
}

// SetDefaultErrorHandler records the error type that is used when this router is the
// root of the schema and no error handler is registered on it. Unlike RegisterErrorHandler,
// it never conflicts with the error type of a parent router.
func (r *RouterRpcSchemas) SetDefaultErrorHandler(fnVal any) error {
	schema, err := BuildGoToTsSchema(errorHandlerMethod, "/", fnVal)
	if err != nil {
		return err
	}

	r.defaultErrorHandler = schema
	return nil
}

// SetTypeOverride makes the generated TypeScript use tsType wherever typ appears.
func (r *RouterRpcSchemas) SetTypeOverride(typ reflect.Type, tsType string) {
	if r.typeOverrides == nil {
		r.typeOverrides = map[reflect.Type]string{}
	}
	r.typeOverrides[typ] = tsType
}

// TypeOverrides returns the type overrides of this router and its mounted children.
// It returns an error when two routers give the same Go type different TypeScript
// types, because the generated file shares one declaration per type.
func (r *RouterRpcSchemas) TypeOverrides() (map[reflect.Type]string, error) {
	overrides := map[reflect.Type]string{}
	if err := r.collectTypeOverrides(overrides, map[*RouterRpcSchemas]bool{}); err != nil {
		return nil, err
	}
	return overrides, nil
}

// collectTypeOverrides adds the overrides of this router and its children to overrides.
// visiting holds the routers on the current mount path, to stop on mount cycles.
func (r *RouterRpcSchemas) collectTypeOverrides(overrides map[reflect.Type]string, visiting map[*RouterRpcSchemas]bool) error {
	if visiting[r] {
		return fmt.Errorf("a router is mounted inside itself")
	}
	visiting[r] = true
	defer delete(visiting, r)

	for typ, tsType := range r.typeOverrides {
		if existing, ok := overrides[typ]; ok && existing != tsType {
			return fmt.Errorf("type %s is registered as both %q and %q", typ, existing, tsType)
		}
		overrides[typ] = tsType
	}

	for _, mount := range r.mounts {
		if err := mount.schemas.collectTypeOverrides(overrides, visiting); err != nil {
			return err
		}
	}
	return nil
}

// Mount adds a child router's schemas under prefix. The child is read when the
// TypeScript is generated, so handlers added to it after mounting are included.
func (r *RouterRpcSchemas) Mount(prefix string, child *RouterRpcSchemas) {
	r.mounts = append(r.mounts, mountedSchemas{prefix: prefix, schemas: child})
}

// errorTypeName describes the error handler's response type for error messages.
func errorTypeName(schema *HandlerSchema) string {
	if schema == nil {
		return "none"
	}
	return schema.returnType.String()
}

// Routes returns every handler of this router and its mounted children, with full URLs.
// The error handler comes first, under the method "ERROR_HANDLER" and the URL "/". It is
// the registered error handler, or else the default one.
// It returns an error when a child router's error handler type differs from this
// router's, because ApiSchema has only one error type, or when routers are mounted in a cycle.
func (r *RouterRpcSchemas) Routes() ([]Route, error) {
	rootError := r.errorHandler
	if rootError == nil {
		rootError = r.defaultErrorHandler
	}

	routes := []Route{}
	if rootError != nil {
		routes = append(routes, newRoute(rootError, rootError.url))
	}

	return r.collectRoutes("", rootError, map[*RouterRpcSchemas]bool{}, routes)
}

// collectRoutes adds the routes of this router and its mounted children to routes,
// with URLs prefixed by prefix. visiting holds the routers on the current mount path.
func (r *RouterRpcSchemas) collectRoutes(prefix string, rootError *HandlerSchema, visiting map[*RouterRpcSchemas]bool, routes []Route) ([]Route, error) {
	if visiting[r] {
		return nil, fmt.Errorf("the router at %q is mounted inside itself", prefix)
	}
	visiting[r] = true
	defer delete(visiting, r)

	for _, schema := range r.schemas {
		routes = append(routes, newRoute(schema, mergePaths(prefix, schema.url)))
	}

	for _, mount := range r.mounts {
		child := mount.schemas
		childPrefix := mergePaths(prefix, mount.prefix)

		if child.errorHandler != nil && (rootError == nil || child.errorHandler.returnType != rootError.returnType) {
			return nil, fmt.Errorf("router at %q has error handler type %s but the root router has %s, and ApiSchema supports only one error type", childPrefix, child.errorHandler.returnType, errorTypeName(rootError))
		}

		var err error
		routes, err = child.collectRoutes(childPrefix, rootError, visiting, routes)
		if err != nil {
			return nil, err
		}
	}

	return routes, nil
}

// newRoute resolves schema to a Route served at fullURL.
func newRoute(schema *HandlerSchema, fullURL string) Route {
	return Route{
		Method:   schema.method,
		URL:      fullURL,
		Params:   schema.paramNames(fullURL),
		Response: schema.returnType,
		Body:     schema.bodyType,
		Query:    schema.queryType,
	}
}

// NewRouterRpcSchemas creates a new RouterRpcSchemas instance with an empty schema collection.
func NewRouterRpcSchemas() *RouterRpcSchemas {
	return &RouterRpcSchemas{
		schemas: []*HandlerSchema{},
	}
}
