package adapter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iaconlabs/transwarp/router"
)

// RunSuiteBenchmarks executes a heavy performance battery to measure the overhead
// of different [router.RouterAdapter] implementations.
func RunSuiteBenchmarks(b *testing.B, factory func() router.Router) {
	// 1. Benchmark: Simple Static Route (Baseline).
	b.Run("Static/Simple", func(b *testing.B) {
		runStaticBenchmark(b, factory())
	})

	// 2. Benchmark: Dynamic Parameters (:id).
	// Measures the cost of value extraction and context injection.
	b.Run("Param/Single", func(b *testing.B) {
		runParamBenchmark(b, factory())
	})

	// 3. Benchmark: "The Onion" (5 levels of Middleware).
	// Measures recursion latency and handler chaining.
	b.Run("Middleware/DeepOnion", func(b *testing.B) {
		runOnionBenchmark(b, factory())
	})
}

func runStaticBenchmark(b *testing.B, adp router.Router) {
	adp.GET("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		adp.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func runParamBenchmark(b *testing.B, adp router.Router) {
	adp.GET("/user/:id", func(_ http.ResponseWriter, r *http.Request) {
		_ = adp.Param(r, "id")
	})
	req := httptest.NewRequest(http.MethodGet, "/user/12345", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		adp.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func runOnionBenchmark(b *testing.B, adp router.Router) {
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	g1 := adp.Group("/g1")
	g1.Use(mw)
	g2 := g1.Group("/g2")
	g2.Use(mw)
	g3 := g2.Group("/g3")
	g3.Use(mw)

	g3.GET("/end", func(_ http.ResponseWriter, _ *http.Request) {}, mw, mw)

	req := httptest.NewRequest(http.MethodGet, "/g1/g2/g3/end", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		adp.ServeHTTP(httptest.NewRecorder(), req)
	}
}
