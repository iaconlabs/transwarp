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


### 2. Basic Examples


#### 2.1 Using Mux (Go  1.22+)

```go
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/profe-ajedrez/transwarp/adapter/muxadapter"
	"github.com/profe-ajedrez/transwarp/server"
)

func main() {
	// 1. Initialize the Standard Mux (Go 1.22+)
	mux := http.NewServeMux()
	
	// 2. Wrap it with the Mux Adapter
	adp := muxadapter.New(mux)

	// 3. Define routes with parameters using the unified API
	adp.GET("/hello/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := adp.Param(r, "name")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, " + name + "!"))
	})

	// 4. Start the managed server with graceful shutdown
	srv := server.New(server.Config{
		Addr: ":8080",
		WriteTimeout: 10 * time.Second,
	}, adp)

	// In a real app, use a context that listens for OS signals
	srv.Start(context.Background())
}
```

#### 2.2 Using Gin

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

| Router                  | Package              | Status        |
| -------------------------| ----------------------| ---------------|
| Gin                     | adapter/ginadapter   | ✅ Stable      |
| Echo                    | adapter/echoadapter  | ✅ Stable      |
| Fiber                   | adapter/fiberadapter | ✅ Stable      |
| Chi                     | adapter/chiadapter   | ✅ Stable      |
| Standard (Mux Go 1.22+) | net/http             | ✅ In Progress |



🧩 Native Go Power

Transwarp isn't just for heavy frameworks. With the Mux Adapter, you can leverage the enhanced routing capabilities introduced in Go 1.22 while maintaining the unified Transwarp API and middleware system. This is perfect for developers who want to keep dependencies to an absolute minimum.


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