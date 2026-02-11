package ginadapter_test

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/profe-ajedrez/transwarp/adapter/ginadapter"
	"github.com/profe-ajedrez/transwarp/server"
)

func TestTranswarp_FullStack_SmokeTest_Gin(t *testing.T) {
	adapter := ginadapter.NewGinAdapter()

	// Ruta con extensión: :id.json
	adapter.GET("/api/v1/users/:id.json", func(w http.ResponseWriter, r *http.Request) {
		id := adapter.Param(r, "id")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","id":"` + id + `"}`))
	})

	srv := server.New(server.Config{Addr: "127.0.0.1:0"}, adapter)
	srvErr := make(chan error, 1)

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	go func() { srvErr <- srv.Start(serverCtx) }()

	addr := srv.Addr()
	client := &http.Client{Timeout: 2 * time.Second}

	// Realizamos la petición
	resp, err := client.Get("http://" + addr + "/api/v1/users/admin.json")
	if err != nil {
		t.Fatalf("Error en la petición: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// Esperamos admin.json (Consistencia Transwarp)
	expected := `{"status":"ok","id":"admin.json"}`
	if string(body) != expected {
		t.Errorf("Gin falló.\nEsperado: %s\nObtenido: %s", expected, string(body))
	}

	// Graceful Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
