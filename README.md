# Transwarp

Transwarp is a lightweight Go library designed to bridge the gap between standard net/http and popular web frameworks. It allows you to write framework-agnostic handlers and middlewares, giving you the freedom to switch between Gin, Echo, Fiber, or Chi without changing your core business logic.


✨ Key Features

  - Framework Agnostic: Write once, run on any supported engine.
  - Low-Cost Abstraction: We tried hard to minimize overhead and memory allocations.
  - Managed Server: Built-in lifecycle management with graceful shutdown support.
  - Unified Middleware: Shared middleware system compatible with all adapters.
  - Hybrid Validation: Integrated binding that merges path parameters and JSON bodies seamlessly.

## 🚀 Quick Start


### 1. Installation

```Bash
go get github.com/profe-ajedrez/transwarp
```


### 2. Basic Example (using Gin)

```Go
package main

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/profe-ajedrez/transwarp/adapter/ginadapter"
	"github.com/profe-ajedrez/transwarp/server"
)

func main() {
	// Initialize your favorite framework
	engine := gin.New()
	
	// Wrap it with a Transwarp Adapter
	adp := ginadapter.New(engine)

	// Define routes using the unified API
	adp.GET("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Transwarp is active!"))
	})

	// Start the managed server
	srv := server.New(server.Config{Addr: ":8080"}, adp)
	srv.Start(context.Background())
}
```


## 🛠️ Supported Adapters

| Router         | Package              | Status        |
| ----------------| ----------------------| ---------------|
| Gin            | adapter/ginadapter   | ✅ Stable      |
| Echo           | adapter/echoadapter  | ✅ Stable      |
| Fiber          | adapter/fiberadapter | ✅ Stable      |
| Chi            | adapter/chiadapter   | ✅ Stable      |
| Standard (Mux) | net/http             | ✅ In Progress |



## 📂 Project Structure

  - /adapter: Internal logic to bridge frameworks.

  - /server: Managed HTTP server with context support.

  - /middleware: Standardized middlewares (Validation, Logging, etc.).

  - /examples: Ready-to-run independent examples for each framework.

## 🧪 Development & Linting

This project follows strict linting rules. To maintain compatibility in this monorepo, we use a patching script:

  - Before running linters

```bash
./patch_mods.sh off
```

  - For local development and running examples

```bash
./patch_mods.sh on
```


## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.