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

// FromEcho returns an HTTP middleware that adapts an Echo middleware into the net/http
// handler chain while preserving shared request state via adapter.TranswarpState.
//
// It ensures a TranswarpState is present in the request context, captures the request
// body before Echo consumes it (so it can be reused), synchronizes Echo path parameters
// back into state.Params, re-injects the captured body for the downstream handler, and
// forwards the next http.Handler via router.NextKey. If the Echo middleware returns an
// error, it is handled by Echo's HTTPErrorHandler.
func FromEcho(echoMw echo.MiddlewareFunc) func(http.Handler) http.Handler {
	e := echo.New()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//fmt.Printf("[DEBUG] Bridge: Petición %s %s recibida\n", r.Method, r.URL.Path)
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

				//fmt.Println("[DEBUG] Bridge: Echo pasó el control al siguiente handler")
				return nil
			}

			// Configuración de Echo Context
			c := e.NewContext(r, w)
			ctx := context.WithValue(r.Context(), router.NextKey, next)
			c.SetRequest(r.WithContext(ctx))

			if err := echoMw(bridgeHandler)(c); err != nil {
				//fmt.Printf("[DEBUG] Bridge: Echo devolvió error: %v\n", err)
				e.HTTPErrorHandler(c, err)
			}
		})
	}
}