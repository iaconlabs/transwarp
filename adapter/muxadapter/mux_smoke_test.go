package muxadapter_test

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/profe-ajedrez/transwarp/adapter/muxadapter"
	"github.com/profe-ajedrez/transwarp/server"
)

func TestTranswarp_FullStack_SmokeTest_Mux(t *testing.T) {
	// 1. Inicializar el adaptador de Mux con el limpiador de puntos
	// Usamos SimpleCleanerMuxConfig porque Go 1.22+ no permite puntos en {nombres}
	cfgMux := muxadapter.SimpleCleanerMuxConfig()
	adapter := muxadapter.NewMuxAdapter(cfgMux)

	// 2. Registrar ruta con extensión
	// Intentamos capturar :id en una ruta que termina en .json
	adapter.GET("/api/v1/users/:id.json", func(w http.ResponseWriter, r *http.Request) {
		id := adapter.Param(r, "id")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","id":"` + id + `"}`))
	})

	// 3. Configurar e iniciar servidor
	srvCfg := server.Config{
		Addr: "127.0.0.1:0", // Puerto dinámico
	}
	srv := server.New(srvCfg, adapter)
	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	go func() {

		_ = srv.Start(serverCtx)
	}()

	addr := srv.Addr()

	// 4. Realizar la petición
	client := &http.Client{Timeout: 2 * time.Second}
	// Probamos admin.json
	resp, err := client.Get("http://" + addr + "/api/v1/users/admin.json")
	if err != nil {
		t.Fatalf("Error en la petición: %v", err)
	}
	defer resp.Body.Close()

	// 5. Validar respuesta
	body, _ := io.ReadAll(resp.Body)

	// NOTA: Aquí es donde Mux suele comportarse distinto a Echo.
	// Si la ruta es /:id.json, Mux captura el segmento completo o solo lo anterior al punto?
	expectedBody := `{"status":"ok","id":"admin.json"}`
	if string(body) != expectedBody {
		t.Errorf("Mux falló. Esperado %s, obtenido %s", expectedBody, string(body))
	}
	// 6. Graceful Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
