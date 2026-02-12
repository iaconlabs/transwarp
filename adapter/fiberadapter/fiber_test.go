package fiberadapter_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/iaconlabs/transwarp"
	"github.com/iaconlabs/transwarp/adapter"
	fiberadapter "github.com/iaconlabs/transwarp/adapter/fiberadapter"
	"github.com/iaconlabs/transwarp/router"
	"github.com/stretchr/testify/assert"
)

func TestAdapter_Compliance(t *testing.T) {
	// Esta prueba confirmará que Echo v5 se comporta como un Driver perfecto
	adapter.RunMuxContract(t, func() router.Router {
		return fiberadapter.NewFiberAdapter()
	})
}

func TestAdapter_Contract(t *testing.T) {
	adapter.RunRouterContract(t, func() router.Router {
		// Usamos la config que maneja los puntos
		return fiberadapter.NewFiberAdapter()
	})
}

func TestAdapter_Advanced(t *testing.T) {
	// Ejecutamos la batería de pruebas de contrato.
	// Cada sub-test recibe una instancia limpia del adaptador.
	adapter.RunAdvancedRouterContract(t, func() router.Router {
		return fiberadapter.NewFiberAdapter()
	})
}

func TestFiberNativeMiddlewareInTranswarp(t *testing.T) {
	// Inicializamos el objeto de aserciones
	is := assert.New(t)

	// 1. Inicializar el adaptador de Fiber v3
	fa := fiberadapter.NewFiberAdapter()

	// 2. Registrar un middleware NATIVO de Fiber
	fa.Use(fiberadapter.FromFiber(func(c fiber.Ctx) error {
		c.Response().Header.Set("X-Fiber-Native", "activated")
		return c.Next()
	}))

	// 3. Crear la instancia de Transwarp
	tw := transwarp.New(fa)

	// 4. Registrar un middleware estándar de Go vía Transwarp
	tw.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Transwarp-Standard", "true")
			next.ServeHTTP(w, r)
		})
	})

	// 5. Definir la ruta
	tw.GET("/middleware-test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("bridge connected"))
	})

	// 6. Ejecutar la petición
	req := httptest.NewRequest(http.MethodGet, "/middleware-test", nil)
	rec := httptest.NewRecorder()

	tw.ServeHTTP(rec, req)

	// --- ASERCIONES ---
	is.Equal("activated", rec.Header().Get("X-Fiber-Native"), "El middleware nativo de Fiber falló")
	is.Equal("true", rec.Header().Get("X-Transwarp-Standard"), "El middleware de Go falló")
	is.Equal(http.StatusOK, rec.Code)
	is.Equal("bridge connected", rec.Body.String())
}

func TestFiberGroupMiddleware(t *testing.T) {
	is := assert.New(t)
	fa := fiberadapter.NewFiberAdapter()
	tw := transwarp.New(fa)

	// 1. Creamos el grupo una sola vez
	v1 := tw.Group("/v1")

	// 2. Registramos el middleware directamente en el grupo de Transwarp
	// Transwarp se encarga de pasarlo al adaptador interno.
	v1.Use(fiberadapter.FromFiber(func(c fiber.Ctx) error {
		// IMPORTANTE: El nombre debe coincidir con el assert de abajo
		c.Response().Header.Set("X-V1-Only", "true")
		return c.Next()
	}))

	// 3. Registramos la ruta en el MISMO grupo
	v1.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("v1 ok"))
	})

	// 4. Ejecutar la petición
	req1 := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	rec1 := httptest.NewRecorder()
	tw.ServeHTTP(rec1, req1)

	// --- ASERCIONES ---
	is.Equal("true", rec1.Header().Get("X-V1-Only"), "El header del middleware de Fiber no llegó")
	is.Equal("v1 ok", rec1.Body.String())

	// 5. Test de aislamiento: Ruta fuera del grupo
	tw.GET("/outside", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("outside"))
	})

	req2 := httptest.NewRequest(http.MethodGet, "/outside", nil)
	rec2 := httptest.NewRecorder()
	tw.ServeHTTP(rec2, req2)

	is.Empty(rec2.Header().Get("X-V1-Only"), "El header de v1 se filtró a una ruta externa")
}

// 1. Test de Continuidad: ¿Fiber permite que el flujo siga a Go?
func TestFromFiber_SuccessFlow(t *testing.T) {
	is := assert.New(t)

	fiberMw := func(c fiber.Ctx) error {
		c.Response().Header.Set("X-Fiber-Step", "processed")
		return c.Next() // Debería llamar al siguiente en la cebolla de Go
	}

	tw := transwarp.New(fiberadapter.NewFiberAdapter())
	tw.Use(fiberadapter.FromFiber(fiberMw))

	tw.GET("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("go-end"))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	tw.ServeHTTP(rec, req)

	is.Equal("processed", rec.Header().Get("X-Fiber-Step"))
	is.Equal("go-end", rec.Body.String())
	is.Equal(http.StatusOK, rec.Code)
}

// 2. Test de Cortocircuito: ¿Fiber puede detener la petición (Auth Fail)?
func TestFromFiber_ShortCircuit(t *testing.T) {
	is := assert.New(t)

	// Middleware que bloquea la petición
	blocker := func(c fiber.Ctx) error {
		return c.Status(http.StatusForbidden).SendString("forbidden-by-fiber")
		// NO llamamos a c.Next()
	}

	tw := transwarp.New(fiberadapter.NewFiberAdapter())
	tw.Use(fiberadapter.FromFiber(blocker))

	handlerReached := false
	tw.GET("/secret", func(w http.ResponseWriter, r *http.Request) {
		handlerReached = true
		w.Write([]byte("top-secret"))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secret", nil)
	tw.ServeHTTP(rec, req)

	is.False(handlerReached, "El handler de Go nunca debió ejecutarse")
	is.Equal(http.StatusForbidden, rec.Code)
	is.Equal("forbidden-by-fiber", rec.Body.String())
}

// 3. Test de Propagación de Parámetros: ¿Go sigue viendo los params después de pasar por Fiber?
func TestFromFiber_ParamIntegrity(t *testing.T) {
	is := assert.New(t)

	// Middleware pasivo de Fiber
	passiveMw := func(c fiber.Ctx) error { return c.Next() }

	tw := transwarp.New(fiberadapter.NewFiberAdapter())
	tw.Use(fiberadapter.FromFiber(passiveMw))

	tw.GET("/user/:id", func(w http.ResponseWriter, r *http.Request) {
		id := tw.Param(r, "id")
		w.Write([]byte("user-" + id))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/user/42", nil)
	tw.ServeHTTP(rec, req)

	is.Equal("user-42", rec.Body.String(), "Los parámetros de ruta se perdieron en el puente")
}

// 4. Test de Orden de Ejecución: Go-MW -> Fiber-MW -> Go-Handler
func TestFromFiber_ExecutionOrder(t *testing.T) {
	is := assert.New(t)
	var order []string

	goMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "go-start")
			next.ServeHTTP(w, r)
			order = append(order, "go-end")
		})
	}

	fiberMw := func(c fiber.Ctx) error {
		order = append(order, "fiber")
		return c.Next()
	}

	tw := transwarp.New(fiberadapter.NewFiberAdapter())
	tw.Use(goMw)
	tw.Use(fiberadapter.FromFiber(fiberMw))

	tw.GET("/order", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	tw.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/order", nil))

	expected := []string{"go-start", "fiber", "handler", "go-end"}
	is.Equal(expected, order, "El orden de la 'cebolla' es incorrecto")
}

// 5. Test de Body: ¿El cuerpo de la petición sobrevive al doble motor?
func TestFromFiber_BodyPersistence(t *testing.T) {
	is := assert.New(t)

	fiberMw := func(c fiber.Ctx) error {
		// Fiber lee el cuerpo
		body := c.Body()
		if string(body) != "ping" {
			return c.Status(400).SendString("bad-body-in-fiber")
		}
		return c.Next()
	}

	tw := transwarp.New(fiberadapter.NewFiberAdapter())
	tw.Use(fiberadapter.FromFiber(fiberMw))

	tw.POST("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
	})

	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader("ping"))
	rec := httptest.NewRecorder()
	tw.ServeHTTP(rec, req)

	is.Equal("ping", rec.Body.String(), "El cuerpo de la petición se corrompió o desapareció")
}
