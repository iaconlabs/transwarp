package muxadapter_test

import (
	"testing"

	"github.com/profe-ajedrez/transwarp/adapter"
	"github.com/profe-ajedrez/transwarp/adapter/muxadapter"
	"github.com/profe-ajedrez/transwarp/router"
)

func BenchmarkMux(b *testing.B) {
	adapter.RunSuiteBenchmarks(b, func() router.Router {
		return muxadapter.NewMuxAdapter(muxadapter.SimpleCleanerMuxConfig())
	})
}
