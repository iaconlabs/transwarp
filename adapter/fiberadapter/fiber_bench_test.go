package fiberadapter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/profe-ajedrez/transwarp"
	"github.com/profe-ajedrez/transwarp/adapter"
	"github.com/profe-ajedrez/transwarp/router"
	"github.com/valyala/fasthttp"
)

func BenchmarkFiber(b *testing.B) {
	adapter.RunSuiteBenchmarks(b, func() router.Router {
		return NewFiberAdapter()
	})
}

// Handler final ultra-rápido
func fastHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// 1. Benchmark: Middleware nativo de Go (Línea base)
func BenchmarkMiddleware_NativeGo(b *testing.B) {
	tw := transwarp.New(NewFiberAdapter())

	// Middleware de Go que no hace nada
	tw.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})

	tw.GET("/bench", fastHandler)
	req := httptest.NewRequest(http.MethodGet, "/bench", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw.ServeHTTP(w, req)
	}
}

// 2. Benchmark: Middleware de Fiber via FromFiber (El "Impuesto")
func BenchmarkMiddleware_FromFiber(b *testing.B) {
	tw := transwarp.New(NewFiberAdapter())

	// Middleware de Fiber que no hace nada
	fiberMw := func(c fiber.Ctx) error {
		return c.Next()
	}

	tw.Use(FromFiber(fiberMw))

	tw.GET("/bench", fastHandler)
	req := httptest.NewRequest(http.MethodGet, "/bench", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw.ServeHTTP(w, req)
	}
}

func BenchmarkMiddleware_FromFiber_DeepStack(b *testing.B) {
	tw := transwarp.New(NewFiberAdapter())

	fiberMw := func(c fiber.Ctx) error {
		return c.Next()
	}

	// Añadimos 5 capas de Fiber
	for i := 0; i < 5; i++ {
		tw.Use(FromFiber(fiberMw))
	}

	tw.GET("/bench", fastHandler)
	req := httptest.NewRequest(http.MethodGet, "/bench", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw.ServeHTTP(w, req)
	}
}

func BenchmarkFiberV3_Native(b *testing.B) {
	app := fiber.New()
	app.Get("/bench", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Usamos el motor directo para una comparativa justa contra fiberadapter
	handler := app.Handler()
	// Simulamos el contexto de fasthttp
	fctx := new(fasthttp.RequestCtx)
	fctx.Request.Header.SetMethod(http.MethodGet)
	fctx.Request.SetRequestURI("/bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler(fctx)
	}
}

func BenchmarkFiberV3_Transwarp(b *testing.B) {
	tw := transwarp.New(NewFiberAdapter())
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
