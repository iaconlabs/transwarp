package ginadapter

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iaconlabs/transwarp/adapter"
	"github.com/iaconlabs/transwarp/router"
)

func FromGin(ginMw gin.HandlerFunc) func(http.Handler) http.Handler {
	engine := gin.New()
	engine.Use(ginMw)
	engine.Any("/*path", func(c *gin.Context) {
		if next, ok := c.Request.Context().Value(router.NextKey).(http.Handler); ok {
			state, ok := c.Request.Context().Value(router.StateKey).(*adapter.TranswarpState)
			if ok {
				for _, p := range c.Params {
					state.Params[p.Key] = p.Value
				}
			}
			next.ServeHTTP(c.Writer, c.Request)
		}
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			state, ok := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
			if !ok {
				state = &adapter.TranswarpState{Params: make(map[string]string)}
				r = r.WithContext(context.WithValue(r.Context(), router.StateKey, state))
			}
			ctx := context.WithValue(r.Context(), router.NextKey, next)
			engine.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
