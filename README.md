# Transwarp

<img width="600" height="auto" alt="image" src="https://github.com/user-attachments/assets/5fb46b99-0269-4c06-9e40-ee2ac3951c06" />


Transwarp is a lightweight Go library designed to bridge the gap between standard net/http and popular web frameworks. It allows you to write framework-agnostic handlers and middlewares, giving you the freedom to switch between Gin, Echo, Fiber, or Chi without changing your core business logic.


✨ Key Features

  - Framework Agnostic: Write once, run on any supported engine.
  - Low-Cost Abstraction: We tried hard to minimize overhead and memory allocations.
  - Managed Server: Built-in lifecycle management with graceful shutdown support.
  - Unified Middleware: Shared middleware system compatible with all adapters.
  - Idiomatic State Management: New helper functions to manage request-scoped data safely.
  - Hybrid Validation: Integrated binding that merges path parameters and JSON bodies seamlessly.


## Check the docs

  - [wiki](https://github.com/iaconlabs/transwarp/wiki)

## Check the examples

  - [Examples folder](examples/)

## 🚀 Quick Start


### 1. Installation

```Bash
go get github.com/iaconlabs/transwarp
```


### 2. State Management (New in v0.0.12)

Transwarp now provides a clean, decoupled way to handle request state (params, user data, etc.) without interacting directly with the context keys.


```go
func MyMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Set a value in the state
        r = transwarp.SetStateValue(r, "tenant_id", "c-99")

        next.ServeHTTP(w, r)
    })
}

func MyHandler(w http.ResponseWriter, r *http.Request) {
    // 2. Retrieve state safely with the comma-ok idiom
    state, ok := transwarp.RequestState(r)
    if ok {
        tenant := state.Params["tenant_id"]
        fmt.Fprint(w, tenant)
    }
}
```


### 3. Basic Examples



#### 3.1 Using Gin

```Go
type UserDTO struct {
	ID    string `param:"id" validate:"required"`
	Name  string `json:"name" validate:"required,min=3"`
	Email string `json:"email" validate:"required,email"`
}

func main() {

	// 2. Wrap it with the Transwarp Adapter.
	adp := ginadapter.NewGinAdapter()

	// 3. Define routes using Transwarp's unified API.
	adp.GET("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Group with validation middleware.
	api := adp.Group("/api/v1")

	api.POST("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		// Retrieve validated data from context.
		user := r.Context().Value("transwarp_val").(*UserDTO)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message": "User %s created", "data": %+v}`, user.ID, user)
	}, middleware.Validate(UserDTO{}))

	// 4. Start the server using Transwarp's managed server.
	srv := server.New(server.Config{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}, adp)

	fmt.Println("Example server running on http://localhost:8080 over  gin")
	if err := srv.Start(context.Background()); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
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

  - `./tools/patch_mods.sh on`: Replaces remote imports with local paths (e.g., replace github.com/iaconlabs/transwarp => ../../).

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

We follow the "Local-First, Remote-Ready" principle. 
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
