package echoadapter_test

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/iaconlabs/transwarp/adapter/echoadapter"
	"github.com/iaconlabs/transwarp/server"
)

func TestTranswarp_FullStack_SmokeTest_Echo(t *testing.T) {
	// 1. Inicializar el adaptador de Echo v5
	adapter := echoadapter.NewEchoAdapter()

	// 2. Registrar una ruta compleja (con parámetros y extensiones)
	// Probamos la capacidad del adaptador de manejar el estado y parámetros
	adapter.GET("/api/v1/users/:id.json", func(w http.ResponseWriter, r *http.Request) {
		// Recuperamos el parámetro usando la abstracción de Transwarp
		id := adapter.Param(r, "id")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","id":"` + id + `"}`))
	})

	// 3. Configurar e iniciar el servidor en un puerto aleatorio (:0)
	cfg := server.Config{
		Addr:         "127.0.0.1:0",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	srv := server.New(cfg, adapter)

	// Canal para capturar errores del servidor
	srvErr := make(chan error, 1)

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	go func() {
		srvErr <- srv.Start(serverCtx)
	}()

	// 4. Obtener la dirección real asignada
	// Gracias al srv.ready chan, esto espera a que el listener esté vivo
	addr := srv.Addr()

	// 5. Realizar la petición HTTP real
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr + "/api/v1/users/admin.json")
	if err != nil {
		t.Fatalf("Error al realizar la petición: %v", err)
	}
	defer resp.Body.Close()

	// 6. Validar la respuesta
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status code esperado 200, obtenido %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	expectedBody := `{"status":"ok","id":"admin.json"}`
	if string(body) != expectedBody {
		t.Errorf("Cuerpo esperado %s, obtenido %s", expectedBody, string(body))
	}

	// 7. Test de Graceful Shutdown (Cierre ordenado)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("Error durante el Shutdown: %v", err)
	}

	// Verificamos que el servidor se detuvo sin errores extraños
	select {
	case err := <-srvErr:
		if err != nil {
			t.Errorf("El servidor falló inesperadamente: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("El servidor tardó demasiado en cerrarse")
	}
}
