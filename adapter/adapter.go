package adapter

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iaconlabs/transwarp/router"
)

const (
	count           = 50
	firstLetterRune = 65 // 'A'
)

// RunMuxContract ejecuta la batería de pruebas específica para drivers basados en net/http.
func RunMuxContract(t *testing.T, factory func() router.Router) {
	t.Run("Parámetros Simples y con Extensiones", func(t *testing.T) {
		// IMPORTANTE: Cada sub-test debe pedir una instancia limpia para evitar pánicos por rutas duplicadas en Mux.
		adapter := factory()

		// Ruta estándar
		adapter.GET("/user/:id", func(w http.ResponseWriter, r *http.Request) {
			id := adapter.Param(r, "id")
			if _, err := w.Write([]byte("id:" + id)); err != nil {
				t.Log(err)
				panic(err)
			}
		})

		// Ruta con extensión (Clave: id.json)
		adapter.GET("/file/:id.json", func(w http.ResponseWriter, r *http.Request) {
			id := adapter.Param(r, "id.json")
			if _, err := w.Write([]byte("file:" + id)); err != nil {
				t.Log(err)
				panic(err)
			}
		})

		adapter.GET("/clients/:id.json", func(w http.ResponseWriter, r *http.Request) {
			id := adapter.Param(r, "id.json")
			if _, err := w.Write([]byte("file:" + id)); err != nil {
				t.Log(err)
				panic(err)
			}
		})

		// Caso 1: ID Simple
		req1 := httptest.NewRequest(http.MethodGet, "/user/123", nil)
		rec1 := httptest.NewRecorder()
		adapter.ServeHTTP(rec1, req1)
		if rec1.Body.String() != "id:123" {
			t.Errorf("Esperado id:123, obtenido %s", rec1.Body.String())
		}

		// Caso 2: ID con Extensión literal
		req2 := httptest.NewRequest(http.MethodGet, "/file/data.json", nil)
		rec2 := httptest.NewRecorder()
		adapter.ServeHTTP(rec2, req2)
		if rec2.Body.String() != "file:data.json" {
			if rec2.Body.String() != "file:data" {
				t.Errorf("Esperado file:data.json, obtenido %s", rec2.Body.String())
				return
			}
			t.Errorf("Esperado file:data.json, obtenido %s", rec2.Body.String())
		}
	})

	t.Run("Jerarquía de Middlewares", func(t *testing.T) {
		adapter := factory()
		result := ""

		mw1 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				result += "A"
				next.ServeHTTP(w, r)
			})
		}

		mw2 := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				result += "B"
				next.ServeHTTP(w, r)
			})
		}

		adapter.Use(mw1) // Global
		adapter.GET("/test-mw", func(_ http.ResponseWriter, _ *http.Request) {
			result += "C"
		}, mw2) // Local de ruta

		req := httptest.NewRequest(http.MethodGet, "/test-mw", nil)
		adapter.ServeHTTP(httptest.NewRecorder(), req)

		// El orden debe ser: Global (A) -> Local (B) -> Handler (C)
		if result != "ABC" {
			t.Errorf("Orden de middlewares incorrecto. Esperado ABC, obtenido %s", result)
		}
	})

	t.Run("Grupos y Prefijos", func(t *testing.T) {
		adapter := factory()
		api := adapter.Group("/api")
		v1 := api.Group("/v1")

		v1.GET("/users", func(w http.ResponseWriter, _ *http.Request) {
			if _, err := w.Write([]byte("users_list")); err != nil {
				t.Log(err)
				panic(err)
			}
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		rec := httptest.NewRecorder()
		adapter.ServeHTTP(rec, req)

		if rec.Body.String() != "users_list" {
			t.Errorf("Error en prefijos de grupo. Obtenido: %s", rec.Body.String())
		}
	})

}

// RunRouterContract executes the core functional contract tests for any [router.Router].
// It ensures consistency in parameter extraction, middleware propagation, and group isolation.
func RunRouterContract(t *testing.T, factory func() router.Router) {
	t.Run("Parameters and Extensions", func(t *testing.T) {
		testParametersAndExtensions(t, factory())
	})

	t.Run("Native Context Propagation", func(t *testing.T) {
		testNativeContextPropagation(t, factory())
	})

	t.Run("Middleware Short-circuit", func(t *testing.T) {
		testMiddlewareShortCircuit(t, factory())
	})

	t.Run("Multiple Complex Parameters", func(t *testing.T) {
		testMultipleComplexParameters(t, factory())
	})

	t.Run("Group and Middleware Isolation", func(t *testing.T) {
		testGroupIsolation(t, factory())
	})

	t.Run("Original Handler Immutability", func(t *testing.T) {
		testHandlerImmutability(t, factory())
	})

	t.Run("Group Union Normalization", func(t *testing.T) {
		testGroupNormalization(t, factory())
	})

	t.Run("Handle and HandleFunc Methods", func(t *testing.T) {
		testHandleAndHandleFunc(t, factory())
	})

	t.Run("ANY Method Multi-registration", func(t *testing.T) {
		testAnyMethod(t, factory())
	})
}

// RunAdvancedRouterContract executes a comprehensive test suite for high-level router features,
// ensuring the implementation handles edge cases like deep nesting, wildcards, and concurrency.
func RunAdvancedRouterContract(t *testing.T, factory func() router.Router) {
	t.Run("Route Priority: Static vs Dynamic", func(t *testing.T) {
		testRoutePriority(t, factory())
	})

	t.Run("Deep Nesting and Onion Middleware", func(t *testing.T) {
		testDeepNestingOnion(t, factory())
	})

	t.Run("Query Params vs Path Params Integrity", func(t *testing.T) {
		testParamsIntegrity(t, factory())
	})

	t.Run("Catch-All Wildcard Routes", func(t *testing.T) {
		testWildcardRoutes(t, factory())
	})

	t.Run("Middleware Status and Header Sync", func(t *testing.T) {
		testHeaderSync(t, factory())
	})

	t.Run("Concurrency Security and Race Conditions", func(t *testing.T) {
		testConcurrency(t, factory())
	})

	t.Run("Middleware Request Body Access", func(t *testing.T) {
		testRequestBodyAccess(t, factory())
	})

	t.Run("Ambiguity Torture Test", func(t *testing.T) {
		testAmbiguity(t, factory())
	})

	t.Run("Custom HTTP Methods via Handle", func(t *testing.T) {
		testCustomMethods(t, factory())
	})
}

func testParametersAndExtensions(t *testing.T, adp router.Router) {
	adp.GET("/user/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain:" + adp.Param(r, "id")))
	})
	adp.GET("/file/:name.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("json:" + adp.Param(r, "name.json")))
	})

	// Case 1: Simple ID
	req1 := httptest.NewRequest(http.MethodGet, "/user/123", nil)
	rec1 := httptest.NewRecorder()
	adp.ServeHTTP(rec1, req1)
	if rec1.Body.String() != "plain:123" {
		t.Errorf("Expected plain:123, got %s", rec1.Body.String())
	}

	// Case 2: Dot in parameter (the dot challenge)
	req2 := httptest.NewRequest(http.MethodGet, "/file/config.json", nil)
	rec2 := httptest.NewRecorder()
	adp.ServeHTTP(rec2, req2)
	if rec2.Body.String() != "json:config.json" {
		t.Errorf("Expected json:config.json, got %s", rec2.Body.String())
	}
}

func testNativeContextPropagation(t *testing.T, adp router.Router) {
	type ctxKey string
	const key ctxKey = "user_id"

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), key, "profe-77")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	adp.Use(mw)
	adp.GET("/profile", func(w http.ResponseWriter, r *http.Request) {
		val, _ := r.Context().Value(key).(string)
		_, _ = w.Write([]byte(val))
	})

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	rec := httptest.NewRecorder()
	adp.ServeHTTP(rec, req)

	if rec.Body.String() != "profe-77" {
		t.Errorf("Context lost. Expected profe-77, got %s", rec.Body.String())
	}
}

func testMiddlewareShortCircuit(t *testing.T, adp router.Router) {
	handlerReached := false
	authMw := func(_ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
		})
	}

	adp.GET("/secret", func(_ http.ResponseWriter, _ *http.Request) {
		handlerReached = true
	}, authMw)

	req := httptest.NewRequest(http.MethodGet, "/secret", nil)
	rec := httptest.NewRecorder()
	adp.ServeHTTP(rec, req)

	if handlerReached {
		t.Error("Handler executed despite middleware abort")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func testMultipleComplexParameters(t *testing.T, adp router.Router) {
	adp.GET("/org/:org_id/repo/:repo_name/files/:path", func(w http.ResponseWriter, r *http.Request) {
		org := adp.Param(r, "org_id")
		repo := adp.Param(r, "repo_name")
		path := adp.Param(r, "path")
		_, _ = w.Write([]byte(org + "|" + repo + "|" + path))
	})

	req := httptest.NewRequest(http.MethodGet, "/org/my.org/repo/my-repo/files/main.go", nil)
	rec := httptest.NewRecorder()
	adp.ServeHTTP(rec, req)

	expected := "my.org|my-repo|main.go"
	if rec.Body.String() != expected {
		t.Errorf("Multiple parameters failed. Expected %s, got %s", expected, rec.Body.String())
	}
}

func testGroupIsolation(t *testing.T, adp router.Router) {
	logs := []string{}
	mwAdmin := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logs = append(logs, "admin")
			next.ServeHTTP(w, r)
		})
	}

	admin := adp.Group("/admin")
	admin.Use(mwAdmin)
	admin.GET("/dashboard", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	public := adp.Group("/public")
	public.GET("/home", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	adp.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/public/home", nil))
	if len(logs) > 0 {
		t.Error("Admin middleware leaked to public group")
	}

	adp.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil))
	if len(logs) != 1 || logs[0] != "admin" {
		t.Error("Admin middleware did not execute correctly")
	}
}

func testHandlerImmutability(t *testing.T, adp router.Router) {
	adp.POST("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(body)
	})

	payload := "hello world"
	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(payload))
	rec := httptest.NewRecorder()
	adp.ServeHTTP(rec, req)

	if rec.Body.String() != payload {
		t.Errorf("Request body corrupted. Expected %s, got %s", payload, rec.Body.String())
	}
}

func testGroupNormalization(t *testing.T, adp router.Router) {
	api := adp.Group("/api/")
	api.GET("/health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	adp.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Group union generated invalid route. Status: %d", rec.Code)
	}
}

func testRoutePriority(t *testing.T, adp router.Router) {
	adp.GET("/post/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("dynamic:" + adp.Param(r, "id")))
	})
	adp.GET("/post/featured", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("static_featured"))
	})

	rec1 := httptest.NewRecorder()
	adp.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/post/featured", nil))
	if rec1.Body.String() != "static_featured" {
		t.Errorf("Priority failure. Expected static route, got: %s", rec1.Body.String())
	}

	rec2 := httptest.NewRecorder()
	adp.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/post/123", nil))
	if rec2.Body.String() != "dynamic:123" {
		t.Errorf("Dynamic route not reached. Got: %s", rec2.Body.String())
	}
}

func testDeepNestingOnion(t *testing.T, adp router.Router) {
	order := ""
	mw := func(tag string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order += "(" + tag
				next.ServeHTTP(w, r)
				order += tag + ")"
			})
		}
	}

	g1 := adp.Group("/g1")
	g1.Use(mw("1"))
	g2 := g1.Group("/g2")
	g2.Use(mw("2"))
	g3 := g2.Group("/g3")
	g3.Use(mw("3"))
	g4 := g3.Group("/g4")
	g4.Use(mw("4"))

	g4.GET("/end", func(_ http.ResponseWriter, _ *http.Request) {
		order += "X"
	}, mw("5"))

	adp.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/g1/g2/g3/g4/end", nil))

	expected := "(1(2(3(4(5X5)4)3)2)1)"
	if order != expected {
		t.Errorf("The 'Onion' hierarchy is incorrect.\nExpected: %s\nGot: %s", expected, order)
	}
}

func testParamsIntegrity(t *testing.T, adp router.Router) {
	adp.GET("/search/:category", func(w http.ResponseWriter, r *http.Request) {
		pathParam := adp.Param(r, "category")
		queryParam := r.URL.Query().Get("q")
		_, _ = w.Write([]byte(pathParam + "|" + queryParam))
	})

	req := httptest.NewRequest(http.MethodGet, "/search/books?q=golang&page=1", nil)
	rec := httptest.NewRecorder()
	adp.ServeHTTP(rec, req)

	if rec.Body.String() != "books|golang" {
		t.Errorf("Parameter collision detected. Got: %s", rec.Body.String())
	}
}

func testWildcardRoutes(t *testing.T, adp router.Router) {
	adp.GET("/static/*path", func(w http.ResponseWriter, r *http.Request) {
		val := adp.Param(r, "path")
		if val == "" {
			val = adp.Param(r, "*")
		}
		_, _ = w.Write([]byte("path:" + val))
	})

	req := httptest.NewRequest(http.MethodGet, "/static/images/logo/brand.png", nil)
	rec := httptest.NewRecorder()
	adp.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "images/logo/brand.png") {
		t.Errorf("Wildcard failed. Got: %s", rec.Body.String())
	}
}

func testHeaderSync(t *testing.T, adp router.Router) {
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	}

	adp.GET("/headers", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Handler", "true")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}, mw)

	rec := httptest.NewRecorder()
	adp.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/headers", nil))

	if rec.Code != http.StatusCreated {
		t.Errorf("Status code lost. Expected 201, got %d", rec.Code)
	}
	if rec.Header().Get("X-Middleware") != "true" || rec.Header().Get("X-Handler") != "true" {
		t.Error("Header synchronization failed in the bridge")
	}
}

func testConcurrency(t *testing.T, adp router.Router) {
	adp.GET("/worker/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(adp.Param(r, "id")))
	})

	results := make(chan bool, count)

	for i := range count {
		go func(val string) {
			req := httptest.NewRequest(http.MethodGet, "/worker/"+val, nil)
			rec := httptest.NewRecorder()
			adp.ServeHTTP(rec, req)
			results <- rec.Body.String() == val
		}(string(rune(i + firstLetterRune)))
	}

	for range count {
		if !<-results {
			t.Error("Concurrency security failure: parameters leaked between parallel requests")
			break
		}
	}
}

func testRequestBodyAccess(t *testing.T, adp router.Router) {
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(strings.NewReader(string(body)))
			next.ServeHTTP(w, r)
		})
	}

	adp.POST("/body", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(body)
	}, mw)

	payload := `{"cmd":"ping"}`
	req := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader(payload))
	rec := httptest.NewRecorder()
	adp.ServeHTTP(rec, req)

	if rec.Body.String() != payload {
		t.Errorf("Body lost after middleware reading. Got: %s", rec.Body.String())
	}
}

func testAmbiguity(t *testing.T, adp router.Router) {
	adp.GET("/a/b/c", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("static")) })
	adp.GET("/a/:b/c", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("param:" + adp.Param(r, "b")))
	})
	adp.GET("/a/*", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("wildcard")) })

	rec1 := httptest.NewRecorder()
	adp.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/a/b/c", nil))
	if rec1.Body.String() != "static" {
		t.Errorf("Ambiguity failure (static). Got: %s", rec1.Body.String())
	}

	rec2 := httptest.NewRecorder()
	adp.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/a/other/c", nil))
	if rec2.Body.String() != "param:other" {
		t.Errorf("Ambiguity failure (param). Got: %s", rec2.Body.String())
	}
}

// 1. Probar Handle con un http.Handler estructurado
type myHandlerForTests struct{}

func (h *myHandlerForTests) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("handler_ok"))
}

func testHandleAndHandleFunc(t *testing.T, adp router.Router) {

	adp.Handle(http.MethodGet, "/test/handle", &myHandlerForTests{})

	// 2. Probar HandleFunc con middleware específico
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Handle", "true")
			next.ServeHTTP(w, r)
		})
	}

	adp.HandleFunc(http.MethodPost, "/test/handlefunc", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("func_ok"))
	}, mw)

	// Ejecución Test 1
	rec1 := httptest.NewRecorder()
	adp.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/test/handle", nil))
	if rec1.Body.String() != "handler_ok" {
		t.Errorf("Handle failed. Expected handler_ok, got %s", rec1.Body.String())
	}

	// Ejecución Test 2
	rec2 := httptest.NewRecorder()
	adp.ServeHTTP(rec2, httptest.NewRequest(http.MethodPost, "/test/handlefunc", nil))
	if rec2.Body.String() != "func_ok" || rec2.Header().Get("X-Handle") != "true" {
		t.Errorf("HandleFunc or Middleware failed. Body: %s, Header: %s",
			rec2.Body.String(), rec2.Header().Get("X-Handle"))
	}
}

func testAnyMethod(t *testing.T, adp router.Router) {
	adp.ANY("/any-route", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("method:" + r.Method))
	})

	methodsToTest := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodOptions,
	}

	for _, method := range methodsToTest {
		req := httptest.NewRequest(method, "/any-route", nil)
		rec := httptest.NewRecorder()
		adp.ServeHTTP(rec, req)

		expected := "method:" + method
		if rec.Body.String() != expected {
			t.Errorf("ANY method failed for %s. Expected %s, got %s", method, expected, rec.Body.String())
		}
	}
}

func testCustomMethods(t *testing.T, adp router.Router) {
	// Muchos ruteadores fallan con métodos no estándar, Transwarp debe soportarlos
	adp.Handle("PURGE", "/cache", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("purged"))
	}))

	req := httptest.NewRequest("PURGE", "/cache", nil)
	rec := httptest.NewRecorder()
	adp.ServeHTTP(rec, req)

	if rec.Body.String() != "purged" {
		t.Errorf("Custom method PURGE failed. Got: %s", rec.Body.String())
	}
}
