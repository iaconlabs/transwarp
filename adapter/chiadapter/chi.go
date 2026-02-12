// Package chiadapter provides the Transwarp implementation for the go-chi framework.
package chiadapter

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/iaconlabs/transwarp/adapter"
	"github.com/iaconlabs/transwarp/router"
)

// ChiAdapter implements router.Router using the chi v5 router.
type ChiAdapter struct {
	mux         *chi.Mux
	prefix      string
	middlewares []func(http.Handler) http.Handler // Middlewares locales al adaptador/grupo
}

// NewChiAdapter initializes a new adapter with an empty chi router.
func NewChiAdapter() *ChiAdapter {
	return &ChiAdapter{
		mux: chi.NewRouter(),
	}
}

// Param extracts parameters from the request context provided by chi.
func (a *ChiAdapter) Param(r *http.Request, key string) string {
	state, ok := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
	if !ok || state.Params == nil {
		return ""
	}
	if val, ok := state.Params[key]; ok {
		return val
	}
	// Fallback Greedy
	cleanKey := key
	if dotIdx := strings.Index(key, "."); dotIdx != -1 {
		cleanKey = key[:dotIdx]
		if val, ok := state.Params[cleanKey]; ok {
			return val
		}
	}
	return ""
}

// ServeHTTP dispatches requests to the chi multiplexer.
func (a *ChiAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	state := &adapter.TranswarpState{Params: make(map[string]string)}

	// Lazy Body inicial
	if r.Body != nil && r.Body != http.NoBody && r.Method != http.MethodGet {
		body, _ := io.ReadAll(r.Body)
		state.Body = body
		r.Body = io.NopCloser(bytes.NewReader(body))
	}

	ctx := context.WithValue(r.Context(), router.StateKey, state)
	a.mux.ServeHTTP(w, r.WithContext(ctx))
}

// Use adds middlewares to the local stack to ensure group isolation.
func (a *ChiAdapter) Use(mws ...func(http.Handler) http.Handler) {
	// IMPORTANTE: Guardamos localmente en lugar de usar a.mux.Use()
	// para mantener el aislamiento de grupos.
	a.middlewares = append(a.middlewares, mws...)
}

// Group returns a new adapter instance for the specified prefix.
func (a *ChiAdapter) Group(prefix string) router.Router {
	// Clonamos los middlewares actuales para el nuevo grupo
	mwsCopy := make([]func(http.Handler) http.Handler, len(a.middlewares))
	copy(mwsCopy, a.middlewares)

	return &ChiAdapter{
		mux:         a.mux,
		prefix:      a.joinPaths(a.prefix, prefix),
		middlewares: mwsCopy,
	}
}

func (a *ChiAdapter) GET(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodGet, p, h, m...)
}

func (a *ChiAdapter) POST(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodPost, p, h, m...)
}

func (a *ChiAdapter) PUT(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodPut, p, h, m...)
}

func (a *ChiAdapter) DELETE(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodDelete, p, h, m...)
}

func (a *ChiAdapter) OPTIONS(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodOptions, p, h, m...)
}

func (a *ChiAdapter) Engine() any { return a.mux }

func (a *ChiAdapter) joinPaths(base, next string) string {
	if next == "" {
		return "/" + strings.Trim(base, "/")
	}
	return strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(next, "/")
}

func (a *ChiAdapter) transformPathForChi(path string) (string, string) {
	wildcardName := ""
	if idx := strings.Index(path, "*"); idx != -1 {
		wildcardName = path[idx+1:]
		if wildcardName == "" {
			wildcardName = "any"
		}
		path = path[:idx] + "*"
	}

	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			paramPart := seg[1:]
			if dotIdx := strings.Index(paramPart, "."); dotIdx != -1 {
				paramPart = paramPart[:dotIdx]
			}
			segments[i] = "{" + paramPart + "}"
		}
	}
	return strings.Join(segments, "/"), wildcardName
}

func (a *ChiAdapter) wrapState(onion http.Handler, wildcardName string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state, ok := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
		if !ok {
			state = &adapter.TranswarpState{Params: make(map[string]string)}
		}

		// Inmutabilidad de parámetros
		newParams := make(map[string]string, len(state.Params))
		for k, v := range state.Params {
			newParams[k] = v
		}

		// Sincronizar parámetros de Chi
		rctx := chi.RouteContext(r.Context())
		if rctx != nil {
			for i, key := range rctx.URLParams.Keys {
				val := rctx.URLParams.Values[i]
				if key == "*" && wildcardName != "" {
					newParams[wildcardName] = val
					newParams["*"] = val
				} else {
					newParams[key] = val
				}
			}
		}

		// Sincronización de Body
		var body []byte = state.Body
		if body == nil && r.Body != nil && r.Body != http.NoBody {
			body, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		newState := &adapter.TranswarpState{Params: newParams, Body: body}
		ctx := context.WithValue(r.Context(), router.StateKey, newState)
		onion.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *ChiAdapter) register(method, path string, h http.HandlerFunc, routeMws ...func(http.Handler) http.Handler) {
	chiPath, wildcardName := a.transformPathForChi(path)
	fullPath := a.joinPaths(a.prefix, chiPath)

	// Construimos la cebolla de middlewares:
	// 1. Middlewares de la ruta específica (los más internos)
	// 2. Middlewares del adaptador/grupo (los más externos)
	var finalHandler http.Handler = h

	// Primero los de la ruta
	for i := len(routeMws) - 1; i >= 0; i-- {
		finalHandler = routeMws[i](finalHandler)
	}

	// Luego los del grupo/globales
	for i := len(a.middlewares) - 1; i >= 0; i-- {
		finalHandler = a.middlewares[i](finalHandler)
	}

	a.mux.Method(method, fullPath, a.wrapState(finalHandler, wildcardName))
}
