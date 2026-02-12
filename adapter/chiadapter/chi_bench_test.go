package chiadapter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/iaconlabs/transwarp"
	"github.com/iaconlabs/transwarp/adapter"
	"github.com/iaconlabs/transwarp/router"
)

func BenchmarkChi(b *testing.B) {
	adapter.RunSuiteBenchmarks(b, func() router.Router {
		return NewChiAdapter()
	})
}

func BenchmarkChi_Native(b *testing.B) {

	r := chi.NewRouter()
	r.Get("/bench", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	req := httptest.NewRequest(http.MethodGet, "/bench", nil)
	w := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkChi_Transwarp(b *testing.B) {
	tw := transwarp.New(NewChiAdapter())
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
