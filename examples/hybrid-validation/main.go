package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/profe-ajedrez/transwarp/adapter/muxadapter"
	"github.com/profe-ajedrez/transwarp/middleware"
	"github.com/profe-ajedrez/transwarp/router"
	"github.com/profe-ajedrez/transwarp/server"
)

// InventoryUpdateDTO demonstrates Transwarp's ability to merge data
// from path parameters and the request body into a single, validated struct.
type InventoryUpdateDTO struct {
	// These fields are extracted from the URL path via the :name syntax.
	// We use PathParamCleaner (SimpleCleaner) to handle dots in filenames if needed.
	WarehouseID string `param:"w_id" validate:"required,uuid"`
	ItemSKU     string `param:"sku" validate:"required,alphanum,len=8"`

	// These fields are extracted from the JSON body.
	Quantity int    `json:"quantity" validate:"required,min=1"`
	Reason   string `json:"reason" validate:"required,max=100"`
}

// main starts an example HTTP server demonstrating hybrid validation that merges URL path parameters and JSON body into an InventoryUpdateDTO.
// It configures a MuxAdapter with SimpleCleaner, registers a POST route at /warehouses/:w_id/items/:sku with validation middleware, prints startup and curl instructions, and runs a managed server on :8080.
func main() {
	// 1. Initialize MuxAdapter with SimpleCleaner to support complex parameter names.
	// This ensures that dots in paths don't break Go's standard ServeMux.
	config := muxadapter.SimpleCleanerMuxConfig()
	adp := muxadapter.NewMuxAdapter(config)

	// 2. Define a route with dynamic parameters using the Transwarp syntax (:param).
	// Your adapter will translate ":w_id" and ":sku" to "{w_id}" and "{sku}" internally.
	route := "/warehouses/:w_id/items/:sku"

	adp.POST(route, handleInventoryUpdate, middleware.Validate(InventoryUpdateDTO{}))

	// 3. Start the managed server.
	srv := server.New(server.Config{
		Addr:         ":8080",
		WriteTimeout: 5 * time.Second,
	}, adp)

	fmt.Println("🛰️ Transwarp Hybrid Validation Example (Mux Edition)")
	fmt.Println("Running on http://localhost:8080")
	fmt.Println("\n📌 Test this with the following CURL:")
	fmt.Println(`curl -X POST http://localhost:8080/warehouses/550e8400-e29b-41d4-a716-446655440000/items/PROD1234 \
     -H "Content-Type: application/json" \
     -d '{"quantity": 50, "reason": "Restock from main hub"}'`)

	if err := srv.Start(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
		os.Exit(1)
	}
}

// handleInventoryUpdate receives a request where data has already been
// On success the response contains keys "status", "message", and "payload".
func handleInventoryUpdate(w http.ResponseWriter, r *http.Request) {
	// Retrieve the validated struct from the context using router.ValidationKey.
	data, ok := r.Context().Value(router.ValidationKey).(*InventoryUpdateDTO)
	if !ok {
		http.Error(w, "Validation data missing in context", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"status":  "ok",
		"message": "Inventory synchronized",
		"payload": data,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}