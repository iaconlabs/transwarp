package muxadapter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"

	"github.com/profe-ajedrez/transwarp/adapter"
	"github.com/profe-ajedrez/transwarp/router"
)

const replazor = "___replazor___"

// PathParamCleaner defines the strategy for encoding/decoding parameter names
// that might contain invalid characters for ServeMux (like dots).
type PathParamCleaner struct {
	encode func(string) string
	decode func(string) string
}

// MuxConfig holds configuration for the ServeMux adapter.
type MuxConfig struct {
	PathParamCleaner PathParamCleaner
}

// NewDefaultMuxConfig returns a configuration with no path cleaning.
func NewDefaultMuxConfig() *MuxConfig {
	return &MuxConfig{PathParamCleaner: PathParamCleaner{
		encode: func(s string) string { return s },
		decode: func(s string) string { return s },
	}}
}

// SimpleCleanerMuxConfig returns a config that replaces dots with a safe placeholder.
func SimpleCleanerMuxConfig() *MuxConfig {
	return &MuxConfig{PathParamCleaner: PathParamCleaner{
		encode: func(s string) string {
			return strings.ReplaceAll(s, ".", replazor)
		},
		decode: func(s string) string {
			return strings.ReplaceAll(s, replazor, ".")
		},
	}}
}

// MuxAdapter implements router.Router using [http.ServeMux].
type MuxAdapter struct {
	mux         *http.ServeMux
	prefix      string
	middlewares []func(http.Handler) http.Handler
	cfg         *MuxConfig
}

// NewMuxAdapter creates a new adapter. If cfg is nil, defaults are used.
func NewMuxAdapter(cfg *MuxConfig) *MuxAdapter {
	if cfg == nil {
		cfg = NewDefaultMuxConfig()
	}
	return &MuxAdapter{
		mux: http.NewServeMux(),
		cfg: cfg,
	}
}

func (a *MuxAdapter) Param(r *http.Request, key string) string {
	state, ok := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
	if !ok || state.Params == nil {
		return ""
	}

	// 1. Coincidencia exacta (ej: "id.json" o "id")
	if val, okKey := state.Params[key]; okKey {
		return val
	}

	// 2. Búsqueda Fuzzy: Si pides "id" pero la llave es "id.json"
	// Esto resuelve el problema de las extensiones en ruteadores rígidos
	for k, v := range state.Params {
		if k == key || strings.HasPrefix(k, key+".") {
			return v
		}
	}

	// 3. Fallback para comodín
	if key == "*" {
		if v, okKey := state.Params["any"]; okKey {
			return v
		}
	}

	return ""
}

func (a *MuxAdapter) GET(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler) {
	a.register(http.MethodGet, path, h, mws...)
}

func (a *MuxAdapter) POST(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler) {
	a.register(http.MethodPost, path, h, mws...)
}

func (a *MuxAdapter) PUT(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler) {
	a.register(http.MethodPut, path, h, mws...)
}

func (a *MuxAdapter) DELETE(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler) {
	a.register(http.MethodDelete, path, h, mws...)
}

func (a *MuxAdapter) Group(prefix string) router.Router {
	mwsCopy := make([]func(http.Handler) http.Handler, len(a.middlewares))
	copy(mwsCopy, a.middlewares)
	return &MuxAdapter{
		mux:         a.mux,
		prefix:      a.joinPaths(a.prefix, prefix),
		middlewares: mwsCopy,
		cfg:         a.cfg,
	}
}

func (a *MuxAdapter) Use(mws ...func(http.Handler) http.Handler) {
	a.middlewares = append(a.middlewares, mws...)
}

func (a *MuxAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Estado inicial para middlewares globales
	state := &adapter.TranswarpState{
		Params: make(map[string]string),
	}
	ctx := context.WithValue(r.Context(), router.StateKey, state)
	a.mux.ServeHTTP(w, r.WithContext(ctx))
}

func (a *MuxAdapter) register(method, path string, h http.HandlerFunc, routeMws ...func(http.Handler) http.Handler) {
	translatedPath, keys := a.translate(path)
	fullPath := a.joinPaths(a.prefix, translatedPath)

	// CRÍTICO: No encodear toda la ruta, solo los tokens internos.
	// El punto literal de ".json" debe quedarse como punto para que Mux haga match.
	pattern := fmt.Sprintf("%s %s", method, fullPath)

	var finalHandler http.Handler = h
	for i := len(routeMws) - 1; i >= 0; i-- {
		finalHandler = routeMws[i](finalHandler)
	}
	for i := len(a.middlewares) - 1; i >= 0; i-- {
		finalHandler = a.middlewares[i](finalHandler)
	}

	a.mux.Handle(pattern, a.wrapState(finalHandler, keys))
}

func (a *MuxAdapter) wrapState(onion http.Handler, keys []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state, _ := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
		if state == nil {
			state = &adapter.TranswarpState{Params: make(map[string]string)}
		}

		newParams := make(map[string]string)
		maps.Copy(newParams, state.Params)

		for _, k := range keys {
			goKey := a.cfg.PathParamCleaner.encode(k)
			if val := r.PathValue(goKey); val != "" {
				newParams[k] = val
			}
		}

		// ... Lógica de Body (Lazy Body) ...
		body := state.Body
		if body == nil && r.Body != nil && r.Body != http.NoBody {
			body, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		newState := &adapter.TranswarpState{Params: newParams, Body: body}
		ctx := context.WithValue(r.Context(), router.StateKey, newState)
		onion.ServeHTTP(w, r.WithContext(ctx))
	})
}

// joinPaths normaliza la unión de prefijos y rutas evitando "//".
func (a *MuxAdapter) joinPaths(base, next string) string {
	if next == "" {
		return "/" + strings.Trim(base, "/")
	}
	// Limpia barras duplicadas en la unión
	return strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(next, "/")
}

func (a *MuxAdapter) Engine() any { return a.mux }

func (a *MuxAdapter) translate(path string) (string, []string) {
	var keys []string

	// 1. Handling Wildcards using strings.Cut for safer parsing.
	if before, after, found := strings.Cut(path, "*"); found {
		name := after
		if name == "" {
			name = "any"
		}
		keys = append(keys, name)

		// USAMOS EL ENCODE: any.json -> any___replazor___json
		safeName := a.cfg.PathParamCleaner.encode(name)
		return before + "{" + safeName + "...}", keys
	}

	// 2. Traducción de parámetros: :id.json -> {id___replazor___json}
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if name, found := strings.CutPrefix(seg, ":"); found {
			keys = append(keys, name)

			// Encode the token for ServeMux internal routing.
			safeName := a.cfg.PathParamCleaner.encode(name)
			segments[i] = "{" + safeName + "}"
		}
	}

	return strings.Join(segments, "/"), keys
}
