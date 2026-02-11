package chiadapter_test

import (
	"testing"

	"github.com/profe-ajedrez/transwarp/adapter"
	"github.com/profe-ajedrez/transwarp/adapter/chiadapter"
	"github.com/profe-ajedrez/transwarp/router"
)

func TestChiAdapter_Compliance(t *testing.T) {
	// Ejecutamos la batería de pruebas de contrato.
	// Cada sub-test recibe una instancia limpia del adaptador.
	adapter.RunMuxContract(t, func() router.Router {
		return chiadapter.NewChiAdapter()
	})
}

func TestMuxAdapter_Contract(t *testing.T) {
	adapter.RunRouterContract(t, func() router.Router {
		// Usamos la config que maneja los puntos
		return chiadapter.NewChiAdapter()
	})
}

func TestChiAdapter_Advanced(t *testing.T) {
	// Ejecutamos la batería de pruebas de contrato.
	// Cada sub-test recibe una instancia limpia del adaptador.
	adapter.RunAdvancedRouterContract(t, func() router.Router {
		return chiadapter.NewChiAdapter()
	})
}
