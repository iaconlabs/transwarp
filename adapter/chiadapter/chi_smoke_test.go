package chiadapter_test

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/iaconlabs/transwarp/adapter/chiadapter"
	"github.com/iaconlabs/transwarp/server"
)

func TestTranswarp_Chi_FullStack_SmokeTest(t *testing.T) {
	// 1. Inicializar el adaptador de Chi
	adapter := chiadapter.NewChiAdapter()

	// 2. Definir una ruta con extensión
	// Chi transformará :id.json -> {id} internamente.
	// Al pedir /admin.json, {id} capturará "admin.json"
	adapter.GET("/api/v1/users/:id.json", func(w http.ResponseWriter, r *http.Request) {
		id := adapter.Param(r, "id")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","id":"` + id + `"}`))
	})

	// 3. Configuración y arranque del servidor
	srv := server.New(server.Config{Addr: "127.0.0.1:0"}, adapter)

	srvErr := make(chan error, 1)

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	go func() {
		srvErr <- srv.Start(serverCtx)
	}()

	addr := srv.Addr()

	// 4. Realizar la petición real
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr + "/api/v1/users/admin.json")
	if err != nil {
		t.Fatalf("Error en la petición: %v", err)
	}
	defer resp.Body.Close()

	// 5. Validar la consistencia (esperamos el valor completo admin.json)
	body, _ := io.ReadAll(resp.Body)
	expectedBody := `{"status":"ok","id":"admin.json"}`
	if string(body) != expectedBody {
		t.Errorf("Chi falló en la consistencia de parámetros. Esperado %s, obtenido %s", expectedBody, string(body))
	}

	// 6. Test de Middleware (Cebolla)
	// Verificamos que los middlewares de Chi inyectados vía adapter funcionen
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("Error en Shutdown: %v", err)
	}

	select {
	case err := <-srvErr:
		if err != nil {
			t.Errorf("Servidor terminó con error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("El servidor de Chi no se detuvo a tiempo")
	}
}
