package fiberadapter

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/iaconlabs/transwarp/adapter"
	"github.com/iaconlabs/transwarp/router"
	"github.com/valyala/fasthttp"
)

var fctxPool = sync.Pool{
	New: func() any { return new(fasthttp.RequestCtx) },
}

func FromFiber(fiberMw fiber.Handler) func(http.Handler) http.Handler {
	engine := fiber.New(fiber.Config{Immutable: true})
	engine.Use(fiberMw)

	engine.All("/*", func(c fiber.Ctx) error {
		next, _ := c.Locals(router.NextKey).(http.Handler)
		c.Locals("tw_flow_continued", true)
		w, _ := c.Locals("tw_writer").(http.ResponseWriter)
		r, _ := c.Locals("tw_req").(*http.Request)

		// 1. Recuperar o inicializar estado
		state, ok := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
		if !ok {
			state = &adapter.TranswarpState{Params: make(map[string]string)}
		}

		// 2. Merge de parámetros (clonando el mapa para evitar efectos secundarios)
		newParams := make(map[string]string)
		for k, v := range state.Params {
			newParams[k] = v
		}

		for _, p := range c.Route().Params {
			rawVal := c.Params(p)
			if rawVal != "" {
				unescaped, _ := url.PathUnescape(rawVal)
				val := clone(unescaped)
				newParams[p] = val
				if strings.HasPrefix(p, "*") {
					newParams["path"] = val
				}
			}
		}

		newState := &adapter.TranswarpState{Params: newParams, Body: state.Body}
		newCtx := context.WithValue(r.Context(), router.StateKey, newState)

		newReq, _ := http.NewRequestWithContext(newCtx, clone(c.Method()), clone(c.OriginalURL()), r.Body)
		newReq.Header = r.Header

		// Sincronizar headers de salida (Fiber -> Go)
		c.Response().Header.VisitAll(func(k, v []byte) {
			key := clone(string(k))
			if w.Header().Get(key) == "" {
				w.Header().Add(key, clone(string(v)))
			}
		})

		next.ServeHTTP(w, newReq)
		return nil
	})

	fastHandler := engine.Handler()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fctx := fctxPool.Get().(*fasthttp.RequestCtx)
			defer fctxPool.Put(fctx)

			fctx.Request.Reset()
			fctx.Response.Reset()

			// 3. Gestión de Estado Inicial y Lazy Body Reading
			state, ok := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
			if !ok {
				var bodyBytes []byte
				if r.Body != nil && r.Body != http.NoBody {
					bodyBytes, _ = io.ReadAll(r.Body)
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
				state = &adapter.TranswarpState{Params: make(map[string]string), Body: bodyBytes}
				r = r.WithContext(context.WithValue(r.Context(), router.StateKey, state))
			}

			if len(state.Body) > 0 {
				fctx.Request.SetBody(state.Body)
			}

			fctx.SetUserValue(router.NextKey, next)
			fctx.SetUserValue("tw_writer", w)
			fctx.SetUserValue("tw_req", r)
			fctx.Request.SetRequestURI(r.URL.RequestURI())
			fctx.Request.Header.SetMethod(r.Method)

			fastHandler(fctx)

			if fctx.UserValue("tw_flow_continued") == nil {
				fctx.Response.Header.VisitAll(func(k, v []byte) {
					w.Header().Set(clone(string(k)), clone(string(v)))
				})
				w.WriteHeader(fctx.Response.StatusCode())
				w.Write(fctx.Response.Body())
			}
		})
	}
}
