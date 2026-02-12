// Package echoadapter provides the Transwarp implementation for the Echo v5 framework.
package echoadapter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/labstack/echo/v5"
	"github.com/profe-ajedrez/transwarp/adapter"
	"github.com/profe-ajedrez/transwarp/router"
)

const defaultMaxShadowCacheSize = 10000

type routeEntry struct {
	method       string
	path         string
	h            http.HandlerFunc
	mws          []func(http.Handler) http.Handler
	regex        *regexp.Regexp
	wildcardName string
}

// EchoAdapter implements router.Router using the Echo v5 framework.
type EchoAdapter struct {
	instance    *echo.Echo
	prefix      string
	middlewares []func(http.Handler) http.Handler
	routes      *[]*routeEntry
	once        *sync.Once

	// Shadow System
	shadowCache     *sync.Map
	shadowCacheSize int32
	maxCacheSize    int
}

// NewEchoAdapter initializes a new adapter with an internal Echo v5 instance.
func NewEchoAdapter() *EchoAdapter {
	e := echo.New()
	return &EchoAdapter{
		instance:     e,
		prefix:       "",
		routes:       &[]*routeEntry{},
		once:         &sync.Once{},
		shadowCache:  &sync.Map{},
		maxCacheSize: defaultMaxShadowCacheSize,
	}
}

// Param retrieves a path parameter, supporting extensions and fuzzy matching.
func (a *EchoAdapter) Param(r *http.Request, key string) string {
	state, ok := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
	if !ok || state.Params == nil {
		return ""
	}

	// 1. Coincidencia exacta (ej: "id" o "file.json")
	if val, ok := state.Params[key]; ok {
		return val
	}

	// 2. Búsqueda inteligente: Si pides "id" pero capturamos "id.json"
	for k, v := range state.Params {
		if k == key || strings.HasPrefix(k, key+".") {
			return v
		}
	}

	// 3. Búsqueda inversa: Si pides "file.json" pero capturamos "file"
	if dotIdx := strings.Index(key, "."); dotIdx != -1 {
		baseKey := key[:dotIdx]
		if val, ok := state.Params[baseKey]; ok {
			return val
		}
	}

	return ""
}

// Group creates a prefixed route group.
func (a *EchoAdapter) Group(prefix string) router.Router {
	return &EchoAdapter{
		instance:     a.instance,
		prefix:       a.joinPaths(a.prefix, prefix),
		middlewares:  append([]func(http.Handler) http.Handler{}, a.middlewares...),
		routes:       a.routes,
		once:         a.once,
		shadowCache:  a.shadowCache,
		maxCacheSize: a.maxCacheSize,
	}
}

// Use registers middlewares into the global stack.
func (a *EchoAdapter) Use(mws ...func(http.Handler) http.Handler) {
	a.middlewares = append(a.middlewares, mws...)
}

// ServeHTTP ensures lazy body reading and context propagation for Echo.
func (a *EchoAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.once.Do(func() { a.registerAll() })

	state := &adapter.TranswarpState{Params: make(map[string]string)}

	// Lazy Body Reading
	if r.Body != nil && r.Body != http.NoBody && r.Method != http.MethodGet {
		body, _ := io.ReadAll(r.Body)
		state.Body = body
		r.Body = io.NopCloser(bytes.NewReader(body))
	}

	ctx := context.WithValue(r.Context(), router.StateKey, state)
	a.instance.ServeHTTP(w, r.WithContext(ctx))
}

func (a *EchoAdapter) GET(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodGet, p, h, m...)
}

func (a *EchoAdapter) POST(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodPost, p, h, m...)
}

func (a *EchoAdapter) PUT(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodPut, p, h, m...)
}

func (a *EchoAdapter) DELETE(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodDelete, p, h, m...)
}

func (a *EchoAdapter) OPTIONS(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodOptions, p, h, m...)
}

func (a *EchoAdapter) Engine() any { return a.instance }

func (a *EchoAdapter) registerAll() {
	shadowZones := make(map[string][]*routeEntry)
	conflictingPrefixes := make(map[string]bool)
	prefixTypes := make(map[string]map[string]bool)

	for _, r := range *a.routes {
		base := a.getStaticBase(r.path)
		if prefixTypes[base] == nil {
			prefixTypes[base] = make(map[string]bool)
		}
		if strings.Contains(r.path, ":") {
			prefixTypes[base][":"] = true
		}
		if strings.Contains(r.path, "*") {
			prefixTypes[base]["*"] = true
		}
	}

	for base, types := range prefixTypes {
		if types[":"] && types["*"] {
			conflictingPrefixes[base] = true
		}
	}

	for _, r := range *a.routes {
		isShadowed := false
		for pref := range conflictingPrefixes {
			if r.path == pref || strings.HasPrefix(r.path, pref+"/") {
				shadowZones[pref] = append(shadowZones[pref], r)
				isShadowed = true
				break
			}
		}
		if !isShadowed {
			// RUTA SEGURA: Registro nativo. wrap ahora recibe el puntero al entry.
			a.instance.Add(r.method, r.path, a.wrap(r))
		}
	}

	for prefix, routes := range shadowZones {
		a.deployShadowRouter(prefix, routes)
	}
}

func (a *EchoAdapter) deployShadowRouter(prefix string, routes []*routeEntry) {
	sort.SliceStable(routes, func(i, j int) bool {
		return a.getRouteScore(routes[i].path) < a.getRouteScore(routes[j].path)
	})

	for _, r := range routes {
		r.regex, r.wildcardName = a.buildRegex(r.path)
	}

	a.instance.Any(prefix+"/*", func(c *echo.Context) error {
		reqPath := c.Request().URL.Path
		method := c.Request().Method
		cacheKey := method + "|" + reqPath

		// 1. Búsqueda en Caché
		if cached, ok := a.shadowCache.Load(cacheKey); ok {
			return a.wrap(cached.(*routeEntry))(c)
		}

		// 2. Búsqueda Lineal
		for _, r := range routes {
			if r.method != method {
				continue
			}
			if r.regex.MatchString(reqPath) {
				// Gestión atómica del caché
				if atomic.AddInt32(&a.shadowCacheSize, 1) > int32(a.maxCacheSize) {
					a.shadowCache = &sync.Map{}
					atomic.StoreInt32(&a.shadowCacheSize, 0)
				}
				a.shadowCache.Store(cacheKey, r)

				return a.wrap(r)(c)
			}
		}
		return echo.ErrNotFound
	})
}

func (a *EchoAdapter) wrap(re *routeEntry) echo.HandlerFunc {
	return func(c *echo.Context) error {
		r := c.Request()
		state, _ := r.Context().Value(router.StateKey).(*adapter.TranswarpState)

		newParams := make(map[string]string)
		for k, v := range state.Params {
			newParams[k] = v
		}

		// 1. Sincronización Nativa (Echo PathValues)
		for _, p := range c.PathValues() {
			newParams[p.Name] = p.Value
		}

		// 2. Sincronización Shadow (Regex)
		// Si el entry tiene regex, significa que estamos en una zona de conflicto
		if re.regex != nil {
			reqPath := r.URL.Path
			matches := re.regex.FindStringSubmatch(reqPath)
			names := re.regex.SubexpNames()

			for i, name := range names {
				if i != 0 && name != "" && i < len(matches) {
					// Revertimos la protección de puntos: _DOT_ -> .
					realName := strings.ReplaceAll(name, "_DOT_", ".")
					newParams[realName] = matches[i]
				}
			}

			// Mapeo de Wildcards para consistencia en Transwarp
			if re.wildcardName != "" {
				val := newParams[re.wildcardName]
				newParams["*"] = val
				newParams["path"] = val
			}
		}

		// 3. Inyección de Estado Consolidado
		newState := &adapter.TranswarpState{
			Params: newParams,
			Body:   state.Body,
		}
		ctx := context.WithValue(r.Context(), router.StateKey, newState)

		// 4. Ejecución de la "Cebolla" (Manual Onion)
		// Construimos la cadena de middlewares y el handler final
		var finalHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			re.h(w, req)
		})

		for i := len(re.mws) - 1; i >= 0; i-- {
			finalHandler = re.mws[i](finalHandler)
		}

		finalHandler.ServeHTTP(c.Response(), r.WithContext(ctx))
		return nil
	}
}

func (a *EchoAdapter) buildRegex(path string) (*regexp.Regexp, string) {
	wildcardName := ""
	// Usamos un placeholder temporal para evitar que QuoteMeta escape nuestros tokens
	p := strings.ReplaceAll(path, ":", "__P__")
	p = strings.ReplaceAll(p, "*", "__W__")
	p = regexp.QuoteMeta(p)

	// Soportamos caracteres alfanuméricos y puntos en el nombre del parámetro
	p = regexp.MustCompile(`__P__([a-zA-Z0-9_.]+)`).ReplaceAllStringFunc(p, func(m string) string {
		name := strings.TrimPrefix(m, "__P__")
		// Go Regex no permite puntos en nombres de grupos. Los normalizamos.
		safeName := strings.ReplaceAll(name, ".", "_DOT_")
		return fmt.Sprintf(`(?P<%s>[^/]+)`, safeName)
	})

	p = regexp.MustCompile(`__W__([a-zA-Z0-9_]*)`).ReplaceAllStringFunc(p, func(m string) string {
		name := strings.TrimPrefix(m, "__W__")
		if name == "" {
			name = "any"
		}
		wildcardName = name
		return fmt.Sprintf(`(?P<%s>.*)`, name)
	})

	return regexp.MustCompile("^" + p + "$"), wildcardName
}

func (a *EchoAdapter) getStaticBase(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") || strings.HasPrefix(p, "*") {
			if i == 0 {
				return "/"
			}
			return strings.Join(parts[:i], "/")
		}
	}
	return "/"
}

func (a *EchoAdapter) getRouteScore(path string) int {
	if strings.Contains(path, "*") {
		return 3
	}
	if strings.Contains(path, ":") {
		return 2
	}
	return 1
}

func (a *EchoAdapter) joinPaths(base, next string) string {
	if next == "" {
		return "/" + strings.Trim(base, "/")
	}
	return strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(next, "/")
}

func (a *EchoAdapter) register(m, p string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler) {
	full := a.joinPaths(a.prefix, p)
	// Registramos la ruta TAL CUAL, permitiendo que Echo v5 maneje sus tokens nativos
	*a.routes = append(*a.routes, &routeEntry{
		method: m,
		path:   full,
		h:      h,
		mws:    append(a.middlewares, mws...),
	})
}
