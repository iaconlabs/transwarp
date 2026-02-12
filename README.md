# Transwarp

Transwarp is a lightweight Go library designed to bridge the gap between standard net/http and popular web frameworks. It allows you to write framework-agnostic handlers and middlewares, giving you the freedom to switch between Gin, Echo, Fiber, or Chi without changing your core business logic.


✨ Key Features

  - Framework Agnostic: Write once, run on any supported engine.
  - Low-Cost Abstraction: We tried hard to minimize overhead and memory allocations.
  - Managed Server: Built-in lifecycle management with graceful shutdown support.
  - Unified Middleware: Shared middleware system compatible with all adapters.
  - Hybrid Validation: Integrated binding that merges path parameters and JSON bodies seamlessly.


## Check the docs

  - [wiki](https://github.com/iaconlabs/transwarp/wiki)

## Check the examples

  - [Examples folder](examples/)

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

| Router                  | Package              | Status   |
| -------------------------| ----------------------| ----------|
| Gin                     | adapter/ginadapter   | ✅ Stable |
| Echo                    | adapter/echoadapter  | ✅ Stable |
| Fiber                   | adapter/fiberadapter | ✅ Stable |
| Chi                     | adapter/chiadapter   | ✅ Stable |
| Standard (Mux Go 1.22+) | net/http             | ✅ Stable |



🧩 Native Go Power

Transwarp isn't just for heavy frameworks. With the Mux Adapter, you can leverage the enhanced routing capabilities introduced in Go 1.22 while maintaining the unified Transwarp API and middleware system. This is perfect for developers who want to keep dependencies to an absolute minimum.


## 📂 Project Structure

  - /adapter: Internal logic to bridge frameworks.

  - /server: Managed HTTP server with context support.

  - /middleware: Standardized middlewares (Validation, Logging, etc.).

  - /examples: Ready-to-run independent examples for each framework.


## Modular Architecture (Multi-Module Monorepo)

Transwarp utilizes a multi-module monorepo strategy where the Core, Adapters, and Middlewares are independent Go modules with their own go.mod. This ensures a minimal dependency footprint and allows for granular versioning. See the [Submodules Wiki] for details on local development using replace directives and our Git tagging convention. 

More about this in the [wiki](https://github.com/iaconlabs/transwarp/wiki/Core-Architecture#submodule-architecture--versioning)


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


## 🛠️ Developer Tooling & Monorepo Workflow

Transwarp is architected as a modular monorepo. This means each adapter and middleware has its own go.mod. To manage this complexity during development, we use a set of specialized tools located in the tools/ directory.

1. ***patch_mods.sh***

The Problem: In Go, if adapter/muxadapter depends on the root transwarp package, it will normally try to download it from GitHub. During development, you want it to use your local (and perhaps unsaved) code.

The Solution: This script toggles replace directives in all go.mod files across the project.

Usage:

  - `./tools/patch_mods.sh on`: Replaces remote imports with local paths (e.g., replace github.com/profe-ajedrez/transwarp => ../../).

  - `./tools/patch_mods.sh off`: Removes the replacements, preparing the code for a clean git commit and remote publishing.

2. ***tagger.sh*** 

The Problem: Since each sub-module has its own versioning, tagging them manually in Git is error-prone. To release version v1.2.3 of the Mux adapter, Git requires a tag named adapter/muxadapter/v1.2.3.

The Solution: This script automates the tagging process for all modules or specific ones, ensuring they follow the required Go monorepo naming convention.

Usage:

  - `./tools/tagger.sh v0.1.0`: Applies the version tag to the root and all sub-modules consistently.

3. ***bundle_code.sh***

The Problem: It's tedious to copy-paste dozens of files while maintaining the directory structure context.

The Solution: This script "flattens" the relevant source code into a single structured document. It preserves file paths and wraps them in Markdown blocks.

Usage:

  - `./tools/bundle_code.sh > project_snapshot.md`: Consolidates the entire codebase into one file.


### 💡 Why we do this

We follow the "Local-First, Remote-Ready" principle.  it.

Note: Always remember to run patch_mods.sh off before pushing to the main branch to ensure the CI/CD pipeline uses clean, remote dependencies.



### Quick Start with Makefile

For convenience, we provide a Makefile that wraps our core tools. This is the recommended way to interact with the project:


| Command                   | Action                           | When to use it?                                                      |     |
| ---------------------------| ----------------------------------| ----------------------------------------------------------------------| -----|
| make dev-on               | Links all sub-modules locally    | Start of work. When you want to see changes across modules instantly |     |
| make dev-off              | Restores remote dependencies     | Before pushing. Ensures your code is clean and passes CI/CD.         |     |
| make bundle               | Generates a single-file snapshot | Code review. When you need to share the full context of the project. |     |
| make release VERSION=v1.x | Tags the entire monorepo         | Deployment. When you are ready to publish a new stable version.      |     |



## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.