package echoadapter

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/profe-ajedrez/transwarp/adapter"
	"github.com/profe-ajedrez/transwarp/router"
)

func FromEcho(echoMw echo.MiddlewareFunc) func(http.Handler) http.Handler {
	e := echo.New()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Recuperar el estado
			state, ok := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
			if !ok {
				state = &adapter.TranswarpState{Params: make(map[string]string)}
				r = r.WithContext(context.WithValue(r.Context(), router.StateKey, state))
			}

			// 2. SEGURIDAD: Lazy Body dentro del bridge
			// Si el estado no tiene el body pero el request sí, lo capturamos antes
			// de que el middleware de Echo lo consuma.
			if state.Body == nil && r.Body != nil && r.Body != http.NoBody && r.Method != http.MethodGet {
				body, _ := io.ReadAll(r.Body)
				state.Body = body
				r.Body = io.NopCloser(bytes.NewReader(body))
			}

			bridgeHandler := func(c *echo.Context) error {
				currentReq := c.Request()

				// Sincronizar parámetros (por si el middleware de Echo los alteró)
				for _, p := range c.PathValues() {
					state.Params[p.Name] = p.Value
				}

				// 3. RE-INYECCIÓN: Restaurar el stream para el siguiente handler
				if state.Body != nil {
					currentReq.Body = io.NopCloser(bytes.NewReader(state.Body))
				}

				if n, ok := currentReq.Context().Value(router.NextKey).(http.Handler); ok {
					n.ServeHTTP(c.Response(), currentReq)
				}
				return nil
			}

			// Configuración de Echo Context
			c := e.NewContext(r, w)
			ctx := context.WithValue(r.Context(), router.NextKey, next)
			c.SetRequest(r.WithContext(ctx))

			if err := echoMw(bridgeHandler)(c); err != nil {
				e.HTTPErrorHandler(c, err)
			}
		})
	}
}
