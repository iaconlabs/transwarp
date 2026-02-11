package echoadapter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/profe-ajedrez/transwarp"
	"github.com/profe-ajedrez/transwarp/adapter"
	"github.com/profe-ajedrez/transwarp/router"
)

func BenchmarkEcho(b *testing.B) {
	adapter.RunSuiteBenchmarks(b, func() router.Router {
		return NewEchoAdapter()
	})
}

// 1. Middleware Nativo de Go
func nativeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Middleware-Type", "native")
		next.ServeHTTP(w, r)
	})
}

// 2. Middleware de Echo v5
func echoMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		c.Response().Header().Set("X-Middleware-Type", "echo")
		return next(c)
	}
}

func BenchmarkMiddlewareOverhead(b *testing.B) {
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Adaptamos el middleware de Echo usando nuestra función
	transwarpEchoMiddleware := FromEcho(echoMiddleware)

	b.Run("Native_Go_Middleware", func(b *testing.B) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		h := nativeMiddleware(finalHandler)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			h.ServeHTTP(w, req)
		}
	})

	b.Run("FromEcho_Bridge_Middleware", func(b *testing.B) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		h := transwarpEchoMiddleware(finalHandler)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			h.ServeHTTP(w, req)
		}
	})
}

func BenchmarkEchoV5_Native(b *testing.B) {
	e := echo.New()
	e.GET("/bench", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	req := httptest.NewRequest(http.MethodGet, "/bench", nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.ServeHTTP(w, req)
	}
}

func BenchmarkEchoV5_Transwarp(b *testing.B) {
	tw := transwarp.New(NewEchoAdapter())
	tw.GET("/bench", transwarpHandler)
	req := httptest.NewRequest(http.MethodGet, "/bench", nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw.ServeHTTP(w, req)
	}
}

func transwarpHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
