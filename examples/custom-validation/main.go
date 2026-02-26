// Package main demonstrates how to extend Transwarp's validation engine
// with custom business logic functions.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/profe-ajedrez/transwarp/adapter/muxadapter"
	"github.com/profe-ajedrez/transwarp/middleware"
	"github.com/profe-ajedrez/transwarp/router"
	"github.com/profe-ajedrez/transwarp/server"
)

// ProductDTO defines a request where the SKU must follow a custom format.
type ProductDTO struct {
	// SKU uses a custom validation tag "sku_format"
	SKU   string  `json:"sku" validate:"required,sku_format"`
	Name  string  `json:"name" validate:"required"`
	Price float64 `json:"price" validate:"required,gt=0"`
}

// skuValidator is our custom business logic.
// skuValidator validates that an SKU string starts with "TW-" (Transwarp).
// It returns true if the field's string value starts with "TW-", false otherwise.
func skuValidator(fl validator.FieldLevel) bool {
	sku := fl.Field().String()
	return strings.HasPrefix(sku, "TW-")
}

// main sets up and runs an example HTTP server that demonstrates a custom validation tag.
// 
// It registers the "sku_format" validation with the underlying validator, mounts a POST
// /products endpoint that validates requests against ProductDTO, prints example curl
// commands, and starts the Transwarp server on :8080.
func main() {
	// 1. Get the underlying validator engine from Transwarp middleware.
	// Transwarp exposes the go-playground/validator instance.
	v := middleware.GetValidator()

	// 2. Register the custom validation tag "sku_format".
	if err := v.RegisterValidation("sku_format", skuValidator); err != nil {
		fmt.Printf("Failed to register validator: %v\n", err)
		return
	}

	// 3. Setup standard Transwarp flow.
	adp := muxadapter.NewMuxAdapter(nil)
	adp.POST("/products", handleCreate, middleware.Validate(ProductDTO{}))

	srv := server.New(server.Config{Addr: ":8080"}, adp)

	fmt.Println("🚀 Custom Validation Example Running")
	fmt.Println("This API requires SKUs to start with 'TW-'")
	fmt.Println("\n📌 Test valid SKU:")
	fmt.Println(`curl -X POST http://localhost:8080/products -d '{"sku":"TW-123", "name":"Adapter", "price":10}'`)
	fmt.Println("\n📌 Test invalid SKU:")
	fmt.Println(`curl -X POST http://localhost:8080/products -d '{"sku":"ABC-123", "name":"Adapter", "price":10}'`)

	srv.Start(context.Background())
}

// handleCreate writes a JSON response containing the created product extracted from the request validation context.
// It reads a *ProductDTO from r.Context().Value(router.ValidationKey), sets Content-Type to application/json, and encodes a response with keys "status" and "product".
func handleCreate(w http.ResponseWriter, r *http.Request) {
	data := r.Context().Value(router.ValidationKey).(*ProductDTO)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "created",
		"product": data,
	})
}