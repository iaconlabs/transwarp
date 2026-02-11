// Package transwarp provides a high-level wrapper to manage web servers
// using interchangeable framework adapters.
package transwarp

import (
	"net/http"

	"github.com/profe-ajedrez/transwarp/router"
)

// Ensure Transwarp implements the RouterAdapter interface.
var _ router.Router = &Transwarp{}

// Transwarp is the primary entry point for the library, wrapping a RouterAdapter
// to provide a consistent API regardless of the underlying web engine.
type Transwarp struct {
	adapter router.Router
}

// New creates a new Transwarp instance using the provided adapter.
func New(adapter router.Router) *Transwarp {
	return &Transwarp{
		adapter: adapter,
	}
}

// ServeHTTP dispatches the request to the underlying adapter.
func (t *Transwarp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.adapter.ServeHTTP(w, r)
}

// GET registers a GET route through the adapter.
func (t *Transwarp) GET(path string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	t.adapter.GET(path, h, m...)
}

// Param retrieves a path parameter through the adapter's logic.
func (t *Transwarp) Param(r *http.Request, key string) string {
	return t.adapter.Param(r, key)
}

// POST registers a POST route through the adapter.
func (t *Transwarp) POST(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler) {
	t.adapter.POST(path, h, mws...)
}

// PUT registers a PUT route through the adapter.
func (t *Transwarp) PUT(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler) {
	t.adapter.PUT(path, h, mws...)
}

// DELETE registers a DELETE route through the adapter.
func (t *Transwarp) DELETE(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler) {
	t.adapter.DELETE(path, h, mws...)
}

// Use adds middlewares to the internal adapter.
func (t *Transwarp) Use(mws ...func(http.Handler) http.Handler) {
	t.adapter.Use(mws...)
}

// Group creates a new prefixed group using the adapter's implementation.
func (t *Transwarp) Group(prefix string) router.Router {
	return t.adapter.Group(prefix)
}

// Engine returns the raw underlying web engine.
func (t *Transwarp) Engine() any {
	return t.adapter.Engine()
}
