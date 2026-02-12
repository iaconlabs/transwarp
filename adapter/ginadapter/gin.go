// Package ginadapter provides the Transwarp implementation for the Gin web framework.
package ginadapter

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

	"github.com/gin-gonic/gin"
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

// GinAdapter implements router.Router using the Gin framework.
type GinAdapter struct {
	engine      *gin.Engine
	prefix      string
	middlewares []func(http.Handler) http.Handler
	routes      *[]*routeEntry
	once        *sync.Once

	shadowCache     *sync.Map
	shadowCacheSize int32
	maxCacheSize    int
}

// NewGinAdapter initializes a new adapter with an internal Gin engine in release mode.
func NewGinAdapter() *GinAdapter {
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	return &GinAdapter{
		engine:       e,
		prefix:       "",
		routes:       &[]*routeEntry{},
		once:         &sync.Once{},
		shadowCache:  &sync.Map{},
		maxCacheSize: defaultMaxShadowCacheSize,
	}
}

// SetMaxShadowCacheSize configures the memory limit for the dynamic shadow router cache.
func (a *GinAdapter) SetMaxShadowCacheSize(size int) {
	a.maxCacheSize = size
}

// Param retrieves a path parameter from the Transwarp state. It supports exact
// matches and smart lookups for parameters with extensions (e.g., :id matching id.json).
func (a *GinAdapter) Param(r *http.Request, key string) string {
	state, ok := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
	if !ok || state.Params == nil {
		return ""
	}

	if val, ok := state.Params[key]; ok {
		return val
	}

	for k, v := range state.Params {
		if strings.HasPrefix(k, key+".") {
			return v
		}
	}

	if dotIdx := strings.Index(key, "."); dotIdx != -1 {
		baseKey := key[:dotIdx]
		if val, ok := state.Params[baseKey]; ok {
			return val
		}
	}

	return ""
}

// Group creates a new route group with a common prefix and inherited middlewares.
func (a *GinAdapter) Group(prefix string) router.Router {
	cleanPrefix := a.prefix + "/" + strings.Trim(prefix, "/")
	cleanPrefix = strings.ReplaceAll(cleanPrefix, "//", "/")

	return &GinAdapter{
		engine:       a.engine,
		prefix:       strings.TrimSuffix(cleanPrefix, "/"),
		middlewares:  append([]func(http.Handler) http.Handler{}, a.middlewares...),
		routes:       a.routes,
		once:         a.once,
		shadowCache:  a.shadowCache,
		maxCacheSize: a.maxCacheSize,
	}
}

// Use adds standard net/http middlewares to the adapter stack.
func (a *GinAdapter) Use(mws ...func(http.Handler) http.Handler) {
	a.middlewares = append(a.middlewares, mws...)
}

// ServeHTTP fulfills the http.Handler interface and handles lazy body reading
// and context synchronization before passing the request to Gin.
func (a *GinAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.once.Do(func() { a.registerAll() })

	state, ok := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
	if !ok {
		state = &adapter.TranswarpState{Params: make(map[string]string)}
	}

	if state.Body == nil && r.Body != nil && r.Body != http.NoBody && r.Method != http.MethodGet {
		body, _ := io.ReadAll(r.Body)
		state.Body = body
		r.Body = io.NopCloser(bytes.NewReader(body))
	}

	ctx := context.WithValue(r.Context(), router.StateKey, state)
	a.engine.ServeHTTP(w, r.WithContext(ctx))
}

// GET registers a GET route.
func (a *GinAdapter) GET(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodGet, p, h, m...)
}

// POST registers a POST route.
func (a *GinAdapter) POST(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodPost, p, h, m...)
}

// PUT registers a PUT route.
func (a *GinAdapter) PUT(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodPut, p, h, m...)
}

// DELETE registers a DELETE route.
func (a *GinAdapter) DELETE(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodDelete, p, h, m...)
}

func (a *GinAdapter) OPTIONS(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodOptions, p, h, m...)
}

// Engine returns the underlying *gin.Engine instance.
func (a *GinAdapter) Engine() any { return a.engine }

// Internal helper methods for route registration and path transformation follow...

func (a *GinAdapter) register(m, p string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler) {
	full := a.prefix + "/" + strings.TrimPrefix(p, "/")
	full = strings.ReplaceAll(full, "//", "/")
	*a.routes = append(*a.routes, &routeEntry{method: m, path: full, h: h, mws: append(a.middlewares, mws...)})
}

func (a *GinAdapter) preparePath(path string) (string, string) {
	if idx := strings.Index(path, "*"); idx != -1 {
		name := path[idx+1:]
		if name == "" {
			name = "any"
		}
		return path[:idx] + "*" + name, name
	}

	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			// :id.json -> :id
			if dotIdx := strings.Index(seg, "."); dotIdx != -1 {
				segments[i] = seg[:dotIdx]
			}
		}
	}
	return strings.Join(segments, "/"), ""
}

func (a *GinAdapter) deployShadowRouter(prefix string, routes []*routeEntry) {
	sort.SliceStable(routes, func(i, j int) bool {
		return a.getRouteScore(routes[i].path) < a.getRouteScore(routes[j].path)
	})

	for _, r := range routes {
		r.regex, r.wildcardName = a.buildRegex(r.path)
	}

	a.engine.Any(prefix+"/*any", func(c *gin.Context) {
		reqPath := c.Request.URL.Path
		method := c.Request.Method
		cacheKey := method + "|" + reqPath

		// 1. Intento de recuperación del caché
		if cached, ok := a.shadowCache.Load(cacheKey); ok {
			a.dispatchWithParams(c, cached.(*routeEntry), reqPath)
			return
		}

		// 2. Despacho por Regex
		for _, r := range routes {
			if r.method != method {
				continue
			}
			if r.regex.MatchString(reqPath) {
				// 3. Almacenamiento con protección de memoria
				currentSize := atomic.LoadInt32(&a.shadowCacheSize)

				if int(currentSize) >= a.maxCacheSize {
					// Purga masiva: si llegamos al límite, reseteamos el mapa.
					// Es un enfoque agresivo pero seguro para evitar OOM.
					a.shadowCache = &sync.Map{}
					atomic.StoreInt32(&a.shadowCacheSize, 0)
				}

				a.shadowCache.Store(cacheKey, r)
				atomic.AddInt32(&a.shadowCacheSize, 1)

				a.dispatchWithParams(c, r, reqPath)
				return
			}
		}
		c.Status(http.StatusNotFound)
	})
}

func (a *GinAdapter) dispatchWithParams(c *gin.Context, r *routeEntry, path string) {
	state, _ := c.Request.Context().Value(router.StateKey).(*adapter.TranswarpState)
	matches := r.regex.FindStringSubmatch(path)

	newParams := make(map[string]string)
	for k, v := range state.Params {
		newParams[k] = v
	}

	names := r.regex.SubexpNames()
	for i, name := range names {
		if i != 0 && name != "" && i < len(matches) {
			realName := strings.ReplaceAll(name, "_DOT_", ".")
			newParams[realName] = matches[i]
		}
	}

	if r.wildcardName != "" {
		val := newParams[r.wildcardName]
		newParams["*"] = val
		newParams["path"] = val
	}

	newState := &adapter.TranswarpState{Params: newParams, Body: state.Body}
	ctx := context.WithValue(c.Request.Context(), router.StateKey, newState)

	var finalHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.h(w, req)
	})

	for i := len(r.mws) - 1; i >= 0; i-- {
		finalHandler = r.mws[i](finalHandler)
	}

	finalHandler.ServeHTTP(c.Writer, c.Request.WithContext(ctx))
}

func (a *GinAdapter) registerInGin(r *routeEntry) {
	ginPath, wcName := a.preparePath(r.path)
	handlers := a.createGinStack(r.h, r.mws, wcName)
	a.engine.Handle(r.method, ginPath, handlers...)
}

func (a *GinAdapter) createGinStack(h http.HandlerFunc, mws []func(http.Handler) http.Handler, wcName string) []gin.HandlerFunc {
	var stack []gin.HandlerFunc

	for _, mw := range mws {
		currentMw := mw
		stack = append(stack, func(c *gin.Context) {
			calledNext := false
			stdNext := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calledNext = true
				c.Request = r
				c.Next()
			})
			currentMw(stdNext).ServeHTTP(c.Writer, c.Request)
			if !calledNext {
				c.Abort()
			}
		})
	}

	stack = append(stack, func(c *gin.Context) {
		if c.IsAborted() {
			return
		}

		state, _ := c.Request.Context().Value(router.StateKey).(*adapter.TranswarpState)
		newParams := make(map[string]string)
		for k, v := range state.Params {
			newParams[k] = v
		}

		for _, p := range c.Params {
			newParams[p.Key] = p.Value
		}

		if wcName != "" {
			val := c.Param(wcName)
			newParams["*"] = val
			newParams["path"] = val
		}

		newState := &adapter.TranswarpState{Params: newParams, Body: state.Body}
		ctx := context.WithValue(c.Request.Context(), router.StateKey, newState)
		h(c.Writer, c.Request.WithContext(ctx))
	})

	return stack
}

func (a *GinAdapter) registerAll() {
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
			a.registerInGin(r)
		}
	}

	for prefix, routes := range shadowZones {
		a.deployShadowRouter(prefix, routes)
	}
}

func (a *GinAdapter) buildRegex(path string) (*regexp.Regexp, string) {
	wildcardName := ""
	p := strings.ReplaceAll(path, ":", "__P__")
	p = strings.ReplaceAll(p, "*", "__W__")
	p = regexp.QuoteMeta(p)

	reParam := regexp.MustCompile(`__P__([a-zA-Z0-9_.]+)`)
	p = reParam.ReplaceAllStringFunc(p, func(m string) string {
		name := strings.TrimPrefix(m, "__P__")
		safeName := strings.ReplaceAll(name, ".", "_DOT_")
		return fmt.Sprintf(`(?P<%s>[^/]+)`, safeName)
	})

	reWild := regexp.MustCompile(`__W__([a-zA-Z0-9_]*)`)
	p = reWild.ReplaceAllStringFunc(p, func(match string) string {
		name := strings.TrimPrefix(match, "__W__")
		if name == "" {
			name = "any"
		}
		wildcardName = name
		return fmt.Sprintf(`(?P<%s>.*)`, name)
	})

	return regexp.MustCompile("^" + p + "$"), wildcardName
}

func (a *GinAdapter) getStaticBase(path string) string {
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

func (a *GinAdapter) getRouteScore(path string) int {
	if strings.Contains(path, "*") {
		return 3
	}
	if strings.Contains(path, ":") {
		return 2
	}
	return 1
}
