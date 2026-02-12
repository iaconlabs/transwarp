package fiberadapter

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/iaconlabs/transwarp/router"
)

// clone fuerza una asignación física de memoria para el string.
func clone(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(s)
	return b.String()
}

// syncParams extrae, decodifica y clona parámetros de Fiber para el contexto de Go.
func syncParams(c fiber.Ctx, baseCtx context.Context) map[string]string {
	newParams := make(map[string]string)

	// 1. Preservar parámetros previos del contexto
	if oldParams, ok := baseCtx.Value(router.ParamsKey).(map[string]string); ok {
		for k, v := range oldParams {
			newParams[k] = v
		}
	}

	// 2. Path Params de Fiber (con Unescape para cumplir con los tests)
	for _, p := range c.Route().Params {
		rawVal := c.Params(p)
		if rawVal != "" {
			unescaped, err := url.PathUnescape(rawVal)
			if err != nil {
				unescaped = rawVal
			}

			val := clone(unescaped)
			newParams[p] = val
			if strings.HasPrefix(p, "*") {
				newParams["path"] = val
				newParams["*"] = val
			}
		}
	}

	// 3. Query Params
	for k, v := range c.Queries() {
		unescaped, _ := url.QueryUnescape(v)
		newParams[clone(k)] = clone(unescaped)
	}

	return newParams
}

// syncHeaders copia las cabeceras de la respuesta de Fiber hacia el ResponseWriter de Go.
func syncHeaders(c fiber.Ctx, w http.ResponseWriter) {
	c.Response().Header.VisitAll(func(k, v []byte) {
		key := clone(string(k))
		if w.Header().Get(key) == "" {
			w.Header().Add(key, clone(string(v)))
		}
	})
}
