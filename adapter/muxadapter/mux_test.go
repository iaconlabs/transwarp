package muxadapter_test

import (
	"testing"

	"github.com/profe-ajedrez/transwarp/adapter"
	"github.com/profe-ajedrez/transwarp/adapter/muxadapter"
	"github.com/profe-ajedrez/transwarp/router"
)

func TestMuxAdapter_Compliance(t *testing.T) {
	// Ejecutamos la suite de contrato pasando la factory del MuxAdapter
	adapter.RunMuxContract(t, func() router.Router {
		return muxadapter.NewMuxAdapter(muxadapter.SimpleCleanerMuxConfig())
	})
}

func TestMuxAdapter_Contract(t *testing.T) {
	adapter.RunRouterContract(t, func() router.Router {
		// Usamos la config que maneja los puntos
		return muxadapter.NewMuxAdapter(muxadapter.SimpleCleanerMuxConfig())
	})
}

func TestChiAdapter_Advanced(t *testing.T) {
	// Ejecutamos la batería de pruebas de contrato.
	// Cada sub-test recibe una instancia limpia del adaptador.
	adapter.RunAdvancedRouterContract(t, func() router.Router {
		return muxadapter.NewMuxAdapter(muxadapter.SimpleCleanerMuxConfig())
	})
}
