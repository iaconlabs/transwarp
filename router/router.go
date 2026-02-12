// Package router defines the core interfaces and context keys used by Transwarp
// to provide a unified abstraction over different web frameworks.
package router

import (
	"net/http"
)

// ctxKey is a private type for context keys to avoid collisions with other packages.
type ctxKey string

const (
	// ParamsKey is used to store and retrieve path parameters from the request context.
	ParamsKey ctxKey = "___transwarp_params___"
	// RouteKey identifies the current route pattern being executed.
	RouteKey ctxKey = "___transwarp_route___"
	// NextKey stores the next [http.Handler] in the middleware chain.
	NextKey ctxKey = "___transwarp_next___"
	// BodyCacheKey is used for internal caching of the request body.
	BodyCacheKey ctxKey = "___transwarp_body_cache___"
	// StateKey provides access to the [TranswarpState] struct containing normalized request data.
	StateKey ctxKey = "___transwarp_state___"
	// ValidationKey is used to store validated data structures after middleware processing.
	ValidationKey ctxKey = "___transwarp_validator_key___"
)

// Router defines the contract that every web framework adapter must implement.
// It allows Transwarp to remain agnostic of the underlying engine (Gin, Echo, Fiber, etc.).
type Router interface {
	// RouterAdapter must satisfy the http.Handler interface.
	http.Handler

	// GET registers a new GET route with optional middlewares.
	GET(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler)
	// POST registers a new POST route with optional middlewares.
	POST(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler)
	// PUT registers a new PUT route with optional middlewares.
	PUT(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler)
	// DELETE registers a new DELETE route with optional middlewares.
	DELETE(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler)
	// OPTIONS registers a new OPTIONS route with optional middlewares.
	OPTIONS(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler)

	ANY(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler)

	Handle(method, path string, h http.Handler, mws ...func(http.Handler) http.Handler)
	HandleFunc(method, path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler)

	// Use adds global middlewares to the adapter.
	Use(mws ...func(http.Handler) http.Handler)
	// Param retrieves a path parameter by its key from the given request.
	Param(r *http.Request, key string) string
	// Group creates a new route group with a common prefix.
	Group(prefix string) Router
	// Engine returns the underlying framework instance (e.g., *gin.Engine).
	Engine() any
}
