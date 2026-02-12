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

// Custom validation logic
func skuValidator(fl validator.FieldLevel) bool {
	sku := fl.Field().String()
	return strings.HasPrefix(sku, "TW-")
}

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

func handleCreate(w http.ResponseWriter, r *http.Request) {
	data := r.Context().Value(router.ValidationKey).(*ProductDTO)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "valid", "data": data})
}
