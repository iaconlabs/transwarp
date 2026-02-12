// Package main demonstrates the ultimate interoperability:
// Using Gin and Echo middlewares inside a Standard Mux project via Transwarp.
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iaconlabs/transwarp/adapter/echoadapter"
	"github.com/iaconlabs/transwarp/adapter/ginadapter"
	"github.com/iaconlabs/transwarp/adapter/muxadapter"
	"github.com/iaconlabs/transwarp/server"
	echoMid "github.com/labstack/echo/v5/middleware"
)

func main() {
	// 1. We start with the lightweight Standard Mux Adapter
	adp := muxadapter.NewMuxAdapter(nil)

	// 2. Wrap Gin's native Logger (Standard Output)
	// Even though we aren't using Gin as a router, we use its logging power.
	ginLogger := ginadapter.FromGin(gin.Logger())
	adp.Use(ginLogger)

	// 3. Wrap Echo's native CORS middleware
	// This handles Preflight requests (OPTIONS) automatically.
	echoCORS := echoadapter.FromEcho(echoMid.CORSWithConfig(echoMid.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}))
	adp.Use(echoCORS)

	// 4. Define a simple route
	adp.GET("/interop", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Logged by Gin, Protected by Echo, Served by Mux!"))
	})

	// 3. Registrar tu ruta de negocio
	adp.OPTIONS("/interop", func(w http.ResponseWriter, r *http.Request) {

	})

	srv := server.New(server.Config{Addr: ":8080"}, adp)

	fmt.Println("🛰️ Transwarp Interop Server Running on :8080")
	fmt.Println("1. Try a normal request: curl -i http://localhost:8080/interop")
	fmt.Println("2. Try a CORS preflight: curl -i -X OPTIONS http://localhost:8080/interop -H 'Origin: http://test.com' -H 'Access-Control-Request-Method: GET'")

	srv.Start(context.Background())
}
