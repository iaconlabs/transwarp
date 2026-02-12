package muxadapter_test

import (
	"testing"

	"github.com/iaconlabs/transwarp/adapter"
	"github.com/iaconlabs/transwarp/adapter/muxadapter"
	"github.com/iaconlabs/transwarp/router"
)

func BenchmarkMux(b *testing.B) {
	adapter.RunSuiteBenchmarks(b, func() router.Router {
		return muxadapter.NewMuxAdapter(muxadapter.SimpleCleanerMuxConfig())
	})
}
