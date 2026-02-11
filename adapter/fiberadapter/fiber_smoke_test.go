package fiberadapter_test

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/profe-ajedrez/transwarp/adapter/fiberadapter"

	"github.com/profe-ajedrez/transwarp/server"
)

func TestTranswarp_FullStack_SmokeTest_Fiber(t *testing.T) {
	// 1. Inicializar el adaptador de Fiber v3
	adapter := fiberadapter.NewFiberAdapter()

	// 2. Definir ruta con extensión
	// transformPathForFiber convertirá esto en /api/v1/users/:id
	adapter.GET("/api/v1/users/:id.json", func(w http.ResponseWriter, r *http.Request) {
		id := adapter.Param(r, "id")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","id":"` + id + `"}`))
	})

	// 3. Configurar e iniciar servidor de Transwarp
	srv := server.New(server.Config{Addr: "127.0.0.1:0"}, adapter)

	srvErr := make(chan error, 1)

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	go func() {
		srvErr <- srv.Start(serverCtx)
	}()

	addr := srv.Addr()

	// 4. Realizar la petición real (net/http -> fasthttp -> net/http)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr + "/api/v1/users/admin.json")
	if err != nil {
		t.Fatalf("Error en la petición: %v", err)
	}
	defer resp.Body.Close()

	// 5. Validar consistencia
	body, _ := io.ReadAll(resp.Body)
	// Fiber captura admin.json completo bajo la llave 'id'
	expectedBody := `{"status":"ok","id":"admin.json"}`
	if string(body) != expectedBody {
		t.Errorf("Fiber falló en la consistencia.\nEsperado: %s\nObtenido: %s", expectedBody, string(body))
	}

	// 6. Graceful Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("Error en Shutdown: %v", err)
	}
}
