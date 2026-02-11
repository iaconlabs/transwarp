package echoadapter_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/profe-ajedrez/transwarp/adapter"
	"github.com/profe-ajedrez/transwarp/adapter/echoadapter"
	"github.com/profe-ajedrez/transwarp/router"
)

func TestEchoV5Adapter_Compliance(t *testing.T) {
	// Esta prueba confirmará que Echo v5 se comporta como un Driver perfecto
	adapter.RunMuxContract(t, func() router.Router {
		return echoadapter.NewEchoAdapter()
	})
}

func TestMuxAdapter_Contract(t *testing.T) {
	adapter.RunRouterContract(t, func() router.Router {
		// Usamos la config que maneja los puntos
		return echoadapter.NewEchoAdapter()
	})
}

func TestChiAdapter_Advanced(t *testing.T) {
	// Ejecutamos la batería de pruebas de contrato.
	// Cada sub-test recibe una instancia limpia del adaptador.
	adapter.RunAdvancedRouterContract(t, func() router.Router {
		return echoadapter.NewEchoAdapter()
	})
}

func TestEchoAdapter_ErrorPropagation(t *testing.T) {
	// 1. Setup
	adapter := echoadapter.NewEchoAdapter()

	// 1. Obtenemos el engine y hacemos el assertion al tipo concreto de Echo v5
	e, ok := adapter.Engine().(*echo.Echo)
	if !ok {
		t.Fatal("El engine no es una instancia de Echo")
	}

	var captured error
	// Configuramos un Error Handler personalizado en Echo para interceptar el error
	e.HTTPErrorHandler = func(c *echo.Context, err error) {
		captured = err
		c.JSON(500, map[string]string{"error": err.Error()})
	}

	// 2. Definimos un Middleware Estándar (nuestro puente)
	mwEstandar := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	// 3. Registramos una ruta que lanza un error nativo de Echo
	// Usamos un Handler que el adaptador envuelve
	adapter.GET("/error", func(_ http.ResponseWriter, _ *http.Request) {
		// Simulamos que algo falló y queremos que Echo lo maneje.
		// En una implementación real, esto podría venir de un middleware nativo de Echo
		// que se ejecutó después de nuestro puente.
	}, mwEstandar)

	// Para forzar un error de Echo DESPUÉS de nuestro middleware,
	// vamos a inyectar un error manualmente en la cadena de Echo.
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			err := next(c)
			if c.Path() == "/error" {
				return echo.NewHTTPError(http.StatusBadRequest, "error_de_prueba")
			}
			return err
		}
	})

	// 4. Ejecución
	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	adapter.ServeHTTP(rec, req)

	// 5. Verificación
	if captured == nil {
		t.Error("FALLO: El middleware estándar silenció (swallowed) el error de Echo.")
	} else if captured.Error() != "code=400, message=error_de_prueba" {
		t.Errorf("Error incorrecto: %v", captured)
	}
}

func TestFromEcho_SuccessFlow(t *testing.T) {
	echoMw := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Response().Header().Set("X-Echo-Bridge", "active")
			return next(c)
		}
	}

	twMw := echoadapter.FromEcho(echoMw)
	finalReached := false
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		finalReached = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	twMw(finalHandler).ServeHTTP(w, req)

	if !finalReached {
		t.Error("El flujo no llegó al handler final de Transwarp")
	}
	if w.Header().Get("X-Echo-Bridge") != "active" {
		t.Error("El middleware de Echo no aplicó los cambios en el Header")
	}
}

func TestFromEcho_ShortCircuit(t *testing.T) {
	echoMw := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Abortamos la petición con un error 403 nativo de Echo
			return echo.NewHTTPError(http.StatusForbidden, "acceso denegado")
		}
	}

	twMw := echoadapter.FromEcho(echoMw)
	finalReached := false
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		finalReached = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	twMw(finalHandler).ServeHTTP(w, req)

	if finalReached {
		t.Error("El short-circuit falló: se alcanzó el handler final")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("Status esperado 403, obtenido %d", w.Code)
	}
}

func TestFromEcho_StateIntegrity(t *testing.T) {
	// Middleware que interactúa con los valores del contexto de Echo
	echoMw := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("echo_val", "foo")
			return next(c)
		}
	}

	twMw := echoadapter.FromEcho(echoMw)

	var capturedState bool
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verificamos que el TranswarpState siga presente en el contexto
		_, ok := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
		capturedState = ok
	})

	state := &adapter.TranswarpState{Params: make(map[string]string)}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Uso directo de context.WithValue siguiendo tu corrección
	ctx := context.WithValue(req.Context(), router.StateKey, state)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	twMw(finalHandler).ServeHTTP(w, req)

	if !capturedState {
		t.Error("El TranswarpState se perdió al pasar por el bridge de Echo")
	}
}

func TestFromEcho_BodyPersistence(t *testing.T) {
	echoMw := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Echo v5 leyendo el body
			body, _ := io.ReadAll(c.Request().Body)
			if string(body) != "payload" {
				return echo.NewHTTPError(http.StatusBadRequest, "body invalido")
			}
			return next(c)
		}
	}

	twMw := echoadapter.FromEcho(echoMw)

	finalReached := false
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Intentamos leerlo de nuevo en Transwarp
		body, _ := io.ReadAll(r.Body)
		if string(body) == "payload" {
			finalReached = true
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("payload"))
	state := &adapter.TranswarpState{Params: make(map[string]string)}
	ctx := context.WithValue(req.Context(), router.StateKey, state)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	twMw(finalHandler).ServeHTTP(w, req)

	if !finalReached {
		t.Error("La persistencia del Body falló tras el bridge")
	}
}

func TestFromEcho_ContextValuePropagation(t *testing.T) {
	type ctxKey string
	const myKey ctxKey = "user-data"

	echoMw := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Echo debería ver el valor del contexto original de Go
			val := c.Request().Context().Value(myKey)
			if val != "secret-info" {
				return echo.NewHTTPError(http.StatusInternalServerError, "contexto perdido en Echo")
			}
			return next(c)
		}
	}

	twMw := echoadapter.FromEcho(echoMw)
	finalReached := false
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(myKey) == "secret-info" {
			finalReached = true
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), myKey, "secret-info")
	req = req.WithContext(ctx)

	twMw(finalHandler).ServeHTTP(httptest.NewRecorder(), req)

	if !finalReached {
		t.Error("El valor del contexto no sobrevivió al viaje de ida y vuelta por Echo")
	}
}

func TestFromEcho_HeaderAndCookieSync(t *testing.T) {
	echoMw := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.SetCookie(&http.Cookie{Name: "session", Value: "12345"})
			c.Response().Header().Set("X-Custom-Header", "Transwarp")
			return next(c)
		}
	}

	twMw := echoadapter.FromEcho(echoMw)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	twMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(w, req)

	if !strings.Contains(w.Header().Get("Set-Cookie"), "session=12345") {
		t.Error("La cookie establecida en Echo no llegó al ResponseWriter")
	}
	if w.Header().Get("X-Custom-Header") != "Transwarp" {
		t.Error("El header establecido en Echo se perdió")
	}
}

func TestFromEcho_ResponseHijacking(t *testing.T) {
	echoMw := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// El middleware escribe y corta el flujo sin llamar a next
			return c.String(http.StatusTeapot, "soy una tetera")
		}
	}

	twMw := echoadapter.FromEcho(echoMw)
	finalReached := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		finalReached = true
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	twMw(handler).ServeHTTP(w, req)

	if finalReached {
		t.Error("El flujo continuó a Transwarp a pesar de que Echo ya había respondido")
	}
	if w.Code != http.StatusTeapot || w.Body.String() != "soy una tetera" {
		t.Errorf("Respuesta incorrecta. Status: %d, Body: %s", w.Code, w.Body.String())
	}
}

func TestFromEcho_Concurrency(t *testing.T) {
	echoMw := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			return next(c)
		}
	}
	twMw := echoadapter.FromEcho(echoMw)

	const workers = 100
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			twMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(w, req)
		}()
	}
	wg.Wait()
}

func TestFromEcho_RequestLoggerIntegration(t *testing.T) {
	// 1. Definimos una estructura para capturar lo que el logger procese
	var capturedStatus int
	var capturedMethod string
	var capturedURI string

	// 2. Configuramos el RequestLogger de Echo v5
	config := middleware.RequestLoggerConfig{
		LogStatus: true,
		LogMethod: true,
		LogURI:    true,
		// Esta es la función que Echo v5 llama al terminar la petición
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			capturedStatus = v.Status
			capturedMethod = v.Method
			capturedURI = v.URI
			return nil
		},
	}

	// Creamos el middleware de Echo v5
	echoMiddleware := middleware.RequestLoggerWithConfig(config)

	// 3. Lo pasamos por nuestro bridge FromEcho
	twMiddleware := echoadapter.FromEcho(echoMiddleware)

	// 4. Handler final de Transwarp que define el status code
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated) // 201
		w.Write([]byte("created"))
	})

	// 5. Ejecutamos la petición
	req := httptest.NewRequest("PUT", "/v5/resource", nil)
	w := httptest.NewRecorder()

	// Ejecutamos la cebolla
	twMiddleware(finalHandler).ServeHTTP(w, req)

	// 6. Validaciones
	if capturedMethod != "PUT" {
		t.Errorf("RequestLogger no capturó el método correcto. Obtenido: %s", capturedMethod)
	}
	if capturedURI != "/v5/resource" {
		t.Errorf("RequestLogger no capturó la URI correcta. Obtenido: %s", capturedURI)
	}
	if capturedStatus != http.StatusCreated {
		t.Errorf("RequestLogger no capturó el status del handler final. Esperado: 201, Obtenido: %d", capturedStatus)
	}
}
