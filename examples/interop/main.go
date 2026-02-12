// Package main demonstrates the ultimate interoperability:
// Using Gin and Echo middlewares inside a Standard Mux project via Transwarp.
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	echoMid "github.com/labstack/echo/v5/middleware"
	"github.com/profe-ajedrez/transwarp/adapter/echoadapter"
	"github.com/profe-ajedrez/transwarp/adapter/ginadapter"
	"github.com/profe-ajedrez/transwarp/adapter/muxadapter"
	"github.com/profe-ajedrez/transwarp/server"
)

// main starts a Transwarp HTTP server that demonstrates using Gin and Echo middlewares
// together with a standard net/http (mux) adapter.
//
// It configures a mux adapter, attaches Gin's logger and Echo's CORS middleware
// (allowing any origin, GET/POST/OPTIONS, and common headers), registers a GET
// handler at /interop that responds with a plain-text message, registers an
// OPTIONS handler to support CORS preflight requests, prints startup instructions,
// and starts the server on :8080.
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