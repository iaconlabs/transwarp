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

type ProductDTO struct {
	// Custom tag "sku_format"
	SKU   string  `json:"sku" validate:"required,sku_format"`
	Name  string  `json:"name" validate:"required"`
	Price float64 `json:"price" validate:"required,gt=0"`
}

// skuValidator reports whether the provided field's string value begins with "TW-".
// It returns true when the value starts with "TW-", false otherwise.
func skuValidator(fl validator.FieldLevel) bool {
	sku := fl.Field().String()
	return strings.HasPrefix(sku, "TW-")
}

// main registers a custom "sku_format" validation that requires SKUs to start with "TW-", configures the POST /products route with validation, creates and starts the HTTP server on ":8080", and prints a startup message.
func main() {
	// 1. Get the shared validator and register our business rule
	v := middleware.GetValidator()
	v.RegisterValidation("sku_format", skuValidator)

	adp := muxadapter.NewMuxAdapter(nil)

	// 2. The middleware will now recognize "sku_format"
	adp.POST("/products", handleCreate, middleware.Validate(ProductDTO{}))

	srv := server.New(server.Config{Addr: ":8080"}, adp)

	fmt.Println("🚀 Custom Validation Active: SKUs must start with 'TW-'")
	srv.Start(context.Background())
}

// handleCreate writes a JSON response indicating validation succeeded and includes the validated ProductDTO.
// It extracts the validated *ProductDTO from the request context using router.ValidationKey and responds with
// Content-Type "application/json" and body `{"status":"valid","data": <product>}`.
func handleCreate(w http.ResponseWriter, r *http.Request) {
	data := r.Context().Value(router.ValidationKey).(*ProductDTO)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "valid", "data": data})
}