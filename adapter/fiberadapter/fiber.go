package fiberadapter

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/profe-ajedrez/transwarp/adapter"
	"github.com/profe-ajedrez/transwarp/router"
	"github.com/valyala/fasthttp"
)

type routeEntry struct {
	method      string
	fullPath    string
	h           http.HandlerFunc
	allHandlers []func(http.Handler) http.Handler
}

type FiberAdapter struct {
	app         *fiber.App
	prefix      string
	middlewares []func(http.Handler) http.Handler
	routes      *[]*routeEntry
	once        *sync.Once
	fastHandler fasthttp.RequestHandler
}

func NewFiberAdapter() *FiberAdapter {
	app := fiber.New(fiber.Config{Immutable: true})
	return &FiberAdapter{
		app:         app,
		prefix:      "",
		middlewares: []func(http.Handler) http.Handler{},
		routes:      &[]*routeEntry{},
		once:        &sync.Once{},
	}
}

func (a *FiberAdapter) Group(prefix string) router.Router {
	cleanPrefix := a.prefix + "/" + strings.Trim(prefix, "/")
	cleanPrefix = strings.ReplaceAll(cleanPrefix, "//", "/")
	mwsCopy := make([]func(http.Handler) http.Handler, len(a.middlewares))
	copy(mwsCopy, a.middlewares)

	return &FiberAdapter{
		app:         a.app,
		prefix:      strings.TrimSuffix(cleanPrefix, "/"),
		middlewares: mwsCopy,
		routes:      a.routes,
		once:        a.once,
	}
}

func (a *FiberAdapter) registerAll() {
	sort.SliceStable(*a.routes, func(i, j int) bool {
		return a.getRouteScore((*a.routes)[i].fullPath) < a.getRouteScore((*a.routes)[j].fullPath)
	})

	for _, r := range *a.routes {
		fiberPath := a.transformPathForFiber(r.fullPath)
		var finalHandler http.Handler = http.HandlerFunc(r.h)
		for i := len(r.allHandlers) - 1; i >= 0; i-- {
			finalHandler = r.allHandlers[i](finalHandler)
		}
		a.app.Add([]string{r.method}, fiberPath, a.wrapAtomic(finalHandler))
	}
	a.fastHandler = a.app.Handler()
}

// Pool global para el adaptador principal
var adapterFctxPool = sync.Pool{
	New: func() any { return new(fasthttp.RequestCtx) },
}

func (a *FiberAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.once.Do(func() { a.registerAll() })

	fctx := adapterFctxPool.Get().(*fasthttp.RequestCtx)

	// Limpieza de seguridad antes de devolver al pool
	defer func() {
		fctx.SetUserValue("tw_ctx", nil)
		fctx.SetUserValue("tw_writer", nil)
		adapterFctxPool.Put(fctx)
	}()

	fctx.Request.Reset()
	fctx.Response.Reset()

	// Es vital pasar el contexto original de Go
	fctx.SetUserValue("tw_ctx", r.Context())
	fctx.SetUserValue("tw_writer", w)

	// Fiber v3 necesita que el URI esté bien formado para el ruteo
	fctx.Request.SetRequestURI(r.URL.RequestURI())
	fctx.Request.Header.SetMethod(r.Method)
	fctx.Request.SetHost(r.Host)

	// Copiamos el cuerpo para que Fiber pueda leerlo
	if r.Body != nil {
		body, _ := io.ReadAll(r.Body)
		fctx.Request.SetBody(body)
		r.Body = io.NopCloser(bytes.NewReader(body))
	}

	fctx.Request.SetRequestURI(r.URL.RequestURI())

	a.fastHandler(fctx)
}

func (a *FiberAdapter) wrapAtomic(onion http.Handler) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctxVal := c.Locals("tw_ctx")
		ctx, ok := ctxVal.(context.Context)
		if !ok || ctx == nil {
			ctx = context.Background()
		}
		w, _ := c.Locals("tw_writer").(http.ResponseWriter)

		// 1. Crear estado único. c.Body() en v3 es seguro con Immutable: true.
		state := &adapter.TranswarpState{
			Params: syncParams(c, ctx),
			Body:   c.Body(),
		}

		// 2. Un solo WithValue para toda la petición
		ctx = context.WithValue(ctx, router.StateKey, state)

		req, _ := http.NewRequestWithContext(
			ctx,
			clone(c.Method()),
			clone(c.OriginalURL()),
			bytes.NewReader(state.Body),
		)

		// Corrección: Acceso al campo Header (fasthttp.RequestHeader)
		c.Request().Header.VisitAll(func(k, v []byte) {
			req.Header.Add(clone(string(k)), clone(string(v)))
		})

		onion.ServeHTTP(w, req)
		return nil
	}
}

type directResponseWriter struct {
	w http.ResponseWriter
}

func (d *directResponseWriter) Header() http.Header         { return d.w.Header() }
func (d *directResponseWriter) Write(p []byte) (int, error) { return d.w.Write(p) }
func (d *directResponseWriter) WriteHeader(s int)           { d.w.WriteHeader(s) }

func (a *FiberAdapter) transformPathForFiber(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			// Limpiamos extensiones: :id.json -> :id
			if dotIdx := strings.Index(seg, "."); dotIdx != -1 {
				segments[i] = seg[:dotIdx]
			}
		} else if strings.HasPrefix(seg, "*") {
			// Fiber v3 solo entiende '*' puro.
			// Convertimos '*path' en '*' y terminamos (el wildcard siempre es el final).
			segments[i] = "*"
			return strings.Join(segments[:i+1], "/")
		}
	}
	return strings.Join(segments, "/")
}

func (a *FiberAdapter) Param(r *http.Request, key string) string {
	state, ok := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
	if !ok || state.Params == nil {
		return ""
	}

	// 1. Intento con la llave exacta (ej: "id")
	if val, ok := state.Params[key]; ok {
		return val
	}

	// 2. FALLBACK CRÍTICO: Si la llave pedida es un wildcard (o se llama "path", "*", "any"),
	// buscamos en la llave genérica "*" que es donde Fiber guarda el catch-all.
	if val, ok := state.Params["*"]; ok {
		return val
	}

	// 3. Fallback para extensiones (id.json -> id)
	cleanKey := key
	if dotIdx := strings.Index(key, "."); dotIdx != -1 {
		cleanKey = key[:dotIdx]
		return state.Params[cleanKey]
	}

	return ""
}

func (a *FiberAdapter) getRouteScore(path string) int {
	if strings.Contains(path, "*") {
		return 3
	}
	if strings.Contains(path, ":") {
		return 2
	}
	return 1
}

func (a *FiberAdapter) Use(mws ...func(http.Handler) http.Handler) {
	a.middlewares = append(a.middlewares, mws...)
}
func (a *FiberAdapter) GET(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodGet, p, h, m...)
}
func (a *FiberAdapter) POST(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodPost, p, h, m...)
}
func (a *FiberAdapter) PUT(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register("PUT", p, h, m...)
}
func (a *FiberAdapter) DELETE(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register("DELETE", p, h, m...)
}

func (a *FiberAdapter) OPTIONS(p string, h http.HandlerFunc, m ...func(http.Handler) http.Handler) {
	a.register(http.MethodOptions, p, h, m...)
}

func (a *FiberAdapter) register(m, p string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler) {
	fullPath := a.prefix + "/" + strings.TrimPrefix(p, "/")
	fullPath = strings.ReplaceAll(fullPath, "//", "/")
	stack := make([]func(http.Handler) http.Handler, len(a.middlewares))
	copy(stack, a.middlewares)
	stack = append(stack, mws...)
	*a.routes = append(*a.routes, &routeEntry{method: m, fullPath: fullPath, h: h, allHandlers: stack})
}
func (a *FiberAdapter) Engine() any { return a.app }
