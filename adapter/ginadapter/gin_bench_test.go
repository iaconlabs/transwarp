package ginadapter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/profe-ajedrez/transwarp"
	"github.com/profe-ajedrez/transwarp/adapter"
	"github.com/profe-ajedrez/transwarp/router"
)

func BenchmarkGin(b *testing.B) {
	adapter.RunSuiteBenchmarks(b, func() router.Router {
		return NewGinAdapter()
	})
}

func BenchmarkGin_ShadowZone(b *testing.B) {
	adp := NewGinAdapter()

	// Forzamos la creación de una Shadow Zone (Colisión)
	adp.GET("/conflict/:id/data", func(w http.ResponseWriter, r *http.Request) {})
	adp.GET("/conflict/*path", func(w http.ResponseWriter, r *http.Request) {})

	// Caso A: Primer Hit (Costo de Regex + Cache Store)
	// Caso B: Hits subsiguientes (Costo de sync.Map.Load)

	req := httptest.NewRequest(http.MethodGet, "/conflict/123/data", nil)
	b.Run("Shadow/FirstMatch", func(b *testing.B) {
		// No reseteamos el timer para incluir el primer proceso
		for i := 0; i < b.N; i++ {
			adp.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
}

func BenchmarkGin_Native(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/bench", func(c *gin.Context) {
		c.String(200, "ok")
	})
	req := httptest.NewRequest(http.MethodGet, "/bench", nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkGin_Transwarp(b *testing.B) {
	tw := transwarp.New(NewGinAdapter())
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
