package server_test

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/iaconlabs/transwarp/router"
	"github.com/iaconlabs/transwarp/server"
)

func TestServer_GracefulShutdown(t *testing.T) {
	// 1. Definimos un handler que tarda 1 segundo en responder
	requestStarted := make(chan struct{})
	requestFinished := make(chan struct{})

	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		time.Sleep(1 * time.Second) // Simula trabajo pesado
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("finalizado"))
		close(requestFinished)
	})

	adapter := &mockAdapter{handler: slowHandler}
	srv := server.New(server.Config{Addr: "127.0.0.1:0"}, adapter) // Puerto 0 para puerto dinámico libre

	// 2. Iniciamos el servidor en una goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start(t.Context())
	}()

	// Esperamos a que el servidor esté realmente escuchando
	addr := srv.Addr()

	// 3. Lanzamos la petición lenta en otra goroutine
	clientResult := make(chan string, 1)
	go func() {
		resp, err := http.Get("http://" + addr)
		if err != nil {
			clientResult <- "error: " + err.Error()
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		clientResult <- string(body)
	}()

	// Esperamos a que la petición llegue al servidor
	<-requestStarted

	// 4. Ejecutamos el Shutdown MIENTRAS la petición está en curso
	shutdownStart := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown falló: %v", err)
	}

	// 5. Verificaciones Finales

	// A. El cliente debió recibir su respuesta completa (no un error de conexión cerrada)
	select {
	case res := <-clientResult:
		if res != "finalizado" {
			t.Errorf("El cliente no recibió la respuesta esperada: %s", res)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout esperando la respuesta del cliente")
	}

	// B. El servidor debió cerrarse sin errores
	select {
	case errServer := <-serverErr:
		if errServer != nil {
			t.Errorf("Start() devolvió un error inesperado: %v", errServer)
		}
	case <-time.After(1 * time.Second):
		t.Error("El servidor no se detuvo después del Shutdown")
	}

	// C. Validar que el tiempo de Shutdown fue razonable (esperó al handler).
	duration := time.Since(shutdownStart)
	if duration < 1*time.Second {
		t.Errorf("El Shutdown fue demasiado rápido (%v), no parece haber esperado al handler", duration)
	}
}

// MockAdapter simple para pruebas de servidor.
type mockAdapter struct {
	handler http.HandlerFunc
}

func (m *mockAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request)                           { m.handler(w, r) }
func (m *mockAdapter) Param(_ *http.Request, _ string) string                                     { return "" }
func (m *mockAdapter) Group(_ string) router.Router                                               { return m }
func (m *mockAdapter) GET(_ string, _ http.HandlerFunc, _ ...func(http.Handler) http.Handler)     {}
func (m *mockAdapter) POST(_ string, _ http.HandlerFunc, _ ...func(http.Handler) http.Handler)    {}
func (m *mockAdapter) PUT(_ string, _ http.HandlerFunc, _ ...func(http.Handler) http.Handler)     {}
func (m *mockAdapter) DELETE(_ string, _ http.HandlerFunc, _ ...func(http.Handler) http.Handler)  {}
func (m *mockAdapter) OPTIONS(_ string, _ http.HandlerFunc, _ ...func(http.Handler) http.Handler) {}
func (m *mockAdapter) Use(_ ...func(http.Handler) http.Handler)                                   {}
func (m *mockAdapter) Engine() any                                                                { return nil }

// Handle registers the handler for the given pattern
func (m *mockAdapter) Handle(method, pattern string, handler http.Handler, mws ...func(http.Handler) http.Handler) {

}

// HandleFunc register an ordinary function as an HTTP handler for a specific path in a web server
func (m *mockAdapter) HandleFunc(method, pattern string, handlerFn http.HandlerFunc, mws ...func(http.Handler) http.Handler) {

}

func (m *mockAdapter) ANY(path string, h http.HandlerFunc, mws ...func(http.Handler) http.Handler) {

}
