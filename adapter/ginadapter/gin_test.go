package ginadapter_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/iaconlabs/transwarp/adapter"
	"github.com/iaconlabs/transwarp/adapter/ginadapter"
	"github.com/iaconlabs/transwarp/router"
)

func TestAdapter_Compliance(t *testing.T) {
	// Ejecutamos la suite de contrato estándar.
	// Pasamos una fábrica que genera un motor de Gin nuevo por cada sub-test.
	adapter.RunMuxContract(t, func() router.Router {
		return ginadapter.NewGinAdapter()
	})
}

func TestAdapter_Contract(t *testing.T) {
	adapter.RunRouterContract(t, func() router.Router {
		// Usamos la config que maneja los puntos
		return ginadapter.NewGinAdapter()
	})
}

func TestAdapter_Advanced(t *testing.T) {
	// Ejecutamos la batería de pruebas de contrato.
	// Cada sub-test recibe una instancia limpia del adaptador.
	adapter.RunAdvancedRouterContract(t, func() router.Router {
		return ginadapter.NewGinAdapter()
	})
}

func TestGinAdapter_Specifics(t *testing.T) {
	t.Run("Requisito de Extensiones :id.json", func(t *testing.T) {
		driver := ginadapter.NewGinAdapter()
		var captured string

		// Registramos la ruta tal como pide el requerimiento
		driver.GET("/user/:id.json", func(w http.ResponseWriter, r *http.Request) {
			captured = driver.Param(r, "id.json") // Gin mapea :id antes del literal .json
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/user/123.json", nil)
		rec := httptest.NewRecorder()
		driver.ServeHTTP(rec, req)

		if captured != "123.json" {
			t.Errorf("Falló la captura del ID con extensión. Esperado 123, obtenido %s", captured)
		}
	})

	t.Run("Inyección de Contexto en Middlewares", func(t *testing.T) {
		driver := ginadapter.NewGinAdapter()

		// Definimos una clave de contexto local y la variable de captura
		type ctxKey string
		const myKey ctxKey = "transwarp_test_key"
		var capturedValue string

		// 1. Middleware estándar (func(http.Handler) http.Handler)
		mw := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Inyectamos un valor en el context nativo
				ctx := context.WithValue(r.Context(), myKey, "pasa_por_el_bridge")
				// IMPORTANTE: Pasamos el nuevo request con el contexto actualizado
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		}

		// 2. Configuramos el driver
		driver.Use(mw)
		driver.GET("/ctx-test", func(w http.ResponseWriter, r *http.Request) {
			// El handler recupera el valor del contexto nativo
			if val, ok := r.Context().Value(myKey).(string); ok {
				capturedValue = val
			}
			w.WriteHeader(http.StatusOK)
		})

		// 3. Ejecutamos la petición
		req := httptest.NewRequest(http.MethodGet, "/ctx-test", nil)
		rec := httptest.NewRecorder()
		driver.ServeHTTP(rec, req)

		// 4. Verificación final
		if capturedValue != "pasa_por_el_bridge" {
			t.Errorf("El contexto se perdió en el bridge. Esperado 'pasa_por_el_bridge', obtenido '%s'", capturedValue)
		}
	})
}

func TestGinAdapter_ShortCircuitProtection(t *testing.T) {
	// 1. Setup
	gin.SetMode(gin.ReleaseMode)
	adapter := ginadapter.NewGinAdapter()

	handlerExecuted := false

	// 2. Definimos un middleware "Portero" (Estándar)
	// Este middleware simula una falla de seguridad: escribe error y corta.
	securityMw := func(_ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("bloqueado por seguridad"))
			// IMPORTANTE: NO llamamos a next.ServeHTTP(w, r)
		})
	}

	// 3. Registramos la ruta protegida
	adapter.GET("/protegido", func(w http.ResponseWriter, _ *http.Request) {
		handlerExecuted = true // Si esto cambia a true, el bridge falló
		w.Write([]byte("información sensible"))
	}, securityMw)

	// 4. Ejecución
	req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
	rec := httptest.NewRecorder()
	adapter.ServeHTTP(rec, req)

	// 5. Verificaciones (Aserciones)
	if handlerExecuted {
		t.Errorf("FALLO DE SEGURIDAD: El handler final se ejecutó a pesar de que el middleware abortó.")
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("Status incorrecto: esperado 403, obtenido %d", rec.Code)
	}

	if rec.Body.String() != "bloqueado por seguridad" {
		t.Errorf("Cuerpo incorrecto: esperado 'bloqueado...', obtenido '%s'", rec.Body.String())
	}
}

func TestGinAdapter_ShadowZone_RaceCondition(t *testing.T) {
	adapter := ginadapter.NewGinAdapter()

	// Registramos rutas que causan una "Shadow Zone" en /conflict
	adapter.GET("/conflict/:id/data", func(w http.ResponseWriter, r *http.Request) {
		id := adapter.Param(r, "id")
		w.Write([]byte("id:" + id))
	})

	adapter.GET("/conflict/*path", func(w http.ResponseWriter, r *http.Request) {
		path := adapter.Param(r, "path")
		w.Write([]byte("path:" + path))
	})

	const iterations = 200
	var wg sync.WaitGroup
	wg.Add(iterations)

	for i := 0; i < iterations; i++ {
		go func(val int) {
			defer wg.Done()

			// Alternamos entre la ruta de parámetro y la de wildcard
			var path, expected string
			if val%2 == 0 {
				path = fmt.Sprintf("/conflict/%d/data", val)
				expected = fmt.Sprintf("id:%d", val)
			} else {
				path = fmt.Sprintf("/conflict/extra/path/%d", val)
				expected = fmt.Sprintf("path:extra/path/%d", val)
			}

			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			adapter.ServeHTTP(rec, req)

			if rec.Body.String() != expected {
				t.Errorf("¡Colisión de datos! URL: %s | Esperado: %s | Obtenido: %s", path, expected, rec.Body.String())
			}
		}(i)
	}

	wg.Wait()
}

func TestFromGin(t *testing.T) {
	// Seteamos Gin en modo Release para evitar logs ruidosos en los tests
	gin.SetMode(gin.ReleaseMode)

	t.Run("Flujo Exitoso (c.Next)", func(t *testing.T) {
		reachedHandler := false

		// 1. Middleware de Gin que simplemente pasa
		ginMw := func(c *gin.Context) {
			c.Header("X-Test-Middleware", "passed")
			c.Next()
		}

		// 2. Adaptamos el middleware
		stdMw := ginadapter.FromGin(ginMw)

		// 3. Handler final
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			reachedHandler = true
			w.WriteHeader(http.StatusOK)
		})

		// 4. Ejecución
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		stdMw(handler).ServeHTTP(rec, req)

		if !reachedHandler {
			t.Error("El handler final no fue alcanzado")
		}
		if rec.Header().Get("X-Test-Middleware") != "passed" {
			t.Error("El middleware de Gin no pudo modificar los headers")
		}
	})

	t.Run("Interrupción de Cadena (c.Abort)", func(t *testing.T) {
		reachedHandler := false

		// 1. Middleware de Gin que aborta (ej: Auth fallido)
		ginMw := func(c *gin.Context) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}

		stdMw := ginadapter.FromGin(ginMw)

		handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			reachedHandler = true
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		stdMw(handler).ServeHTTP(rec, req)

		if reachedHandler {
			t.Error("El handler final fue alcanzado pero el middleware llamó a c.Abort()")
		}
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Esperado status 401, obtenido %d", rec.Code)
		}
	})

	t.Run("Propagación de Contexto Nativo", func(t *testing.T) {
		type ctxKey string
		const key ctxKey = "user"
		var capturedUser string

		// 1. Middleware de Gin corregido
		ginMw := func(c *gin.Context) {
			// Leemos del contexto nativo para validar que el bridge funciona
			user, ok := c.Request.Context().Value(key).(string)
			if !ok || user == "" {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			// Inyectamos un valor nuevo "hacia adelante"
			ctx := context.WithValue(c.Request.Context(), ctxKey("role"), "admin")
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		}

		stdMw := ginadapter.FromGin(ginMw)

		// 2. Handler final que captura ambos valores
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedUser = r.Context().Value(key).(string)
			role, _ := r.Context().Value(ctxKey("role")).(string)
			w.Write([]byte(role))
		})

		// 3. Configuración del request con contexto inicial
		rootCtx := context.WithValue(context.Background(), key, "profe-ajedrez")
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(rootCtx)
		rec := httptest.NewRecorder()

		stdMw(handler).ServeHTTP(rec, req)

		// 4. Aserciones
		if capturedUser != "profe-ajedrez" {
			t.Errorf("El contexto original se perdió. Obtenido: %s", capturedUser)
		}
		if rec.Body.String() != "admin" {
			t.Error("El contexto modificado por el middleware de Gin no llegó al handler")
		}
	})

	t.Run("Seguridad en Concurrencia (Race Condition Check)", func(t *testing.T) {
		// Este test asegura que el 'next' de una petición no se mezcle con el de otra
		ginMw := func(c *gin.Context) { c.Next() }
		stdMw := ginadapter.FromGin(ginMw)

		const iterations = 100
		var wg sync.WaitGroup
		wg.Add(iterations)

		for i := range iterations {
			go func(val int) {
				defer wg.Done()

				// Cada petición espera un valor distinto en el body
				expected := string(rune(val))
				handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Write([]byte(expected))
				})

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				rec := httptest.NewRecorder()
				stdMw(handler).ServeHTTP(rec, req)

				if rec.Body.String() != expected {
					t.Errorf("Mezcla de peticiones detectada: esperado %s, obtenido %s", expected, rec.Body.String())
				}
			}(i)
		}
		wg.Wait()
	})
}
