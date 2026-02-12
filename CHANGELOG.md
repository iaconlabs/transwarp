# Changelog

All notable changes to the Transwarp project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.


[Unreleased] - 2026-02-11

Added

  - First-class OPTIONS Support: Added the OPTIONS method to the Router interface and implemented it across all adapters (MuxAdapter, GinAdapter, FiberAdapter). This ensures that preflight requests can be handled natively within the Transwarp middleware pipeline.

  - Middleware Interoperability Example: New comprehensive example in examples/middleware-interop demonstrating a "Triple-Bridge" architecture:

    * Core: Standard net/http (Mux).

    * Logging: Native Gin Logger.

    * CORS: Native Echo v5 CORS middleware.

  - Cross-Framework Bridge Documentation: Added specialized Wiki pages covering the FromGin, FromEcho, and FromFiber conversion logic and lifecycle.

  - First-class OPTIONS support: Added the OPTIONS method to the Router interface and implemented it across all adapters (MuxAdapter, GinAdapter, FiberAdapter).

  - Developer Toolkit: Added a suite of automation scripts in tools/ for monorepo management:

      - patch_mods.sh: Manages local replace directives for seamless cross-module development.

      - tagger.sh: Automates semantic versioning for multiple modules simultaneously.

      - bundle_code.sh: Consolidates the codebase into a single document for review or AI context.

  - Centralized Makefile: Implemented a root Makefile to provide a unified interface for development, testing, and benchmarking.

  - Dynamic Test Flags: Support for passing custom arguments to tests and benchmarks via the ARGS variable (e.g., make test-all ARGS="-v").

Changed

  - MuxAdapter Pattern Matching: Refactored the internal register method in MuxAdapter to strictly follow the Go 1.22+ routing patterns ("METHOD PATH"). This prevents route collisions and allows distinct handling for GET, POST, and OPTIONS on the same path.

  - Echo Adapter (v5): Updated echoadapter to support Echo v5's context and middleware signatures, ensuring compatibility with the latest generation of the framework.

  - MuxAdapter Pattern Matching: Updated internal registration to use Go 1.22+ strict method patterns ("METHOD PATH"), preventing route collisions.

  - Modular Architecture: Refactored the repository into a Multi-Module Monorepo to ensure a minimal dependency footprint for end-users.

Fixed

  - CORS Preflight Bypass: Resolved a critical issue where OPTIONS requests were bypassing the middleware chain in the MuxAdapter. By requiring explicit OPTIONS registration via the adapter, preflight requests now correctly trigger bridge middlewares (like Echo CORS) before reaching the business logic.

  - Lazy Body Capture in Bridges: Improved the FromEcho bridge to implement "Lazy Body Reading," preventing middleware from exhausting the request stream before it reaches the final Transwarp handler.

  - CORS Preflight Bypass: Fixed a critical bug where OPTIONS requests were bypassing the middleware chain in the MuxAdapter.

  - Echo v5 Interoperability: Corrected the FromEcho bridge to properly handle context termination and status codes during CORS preflight.



[prerelease] - 2026-02-10

Added

  - Validation Extensibility: Introduced middleware.GetValidator() to expose the shared validator.Validate instance. This allows users to register custom validation tags and business rules.

  - Hybrid Validation Example: Added examples/hybrid-validation demonstrating merged data binding from URL paths and JSON bodies.

  - Triple-Binding Example: Added examples/full-binding showcasing simultaneous binding of Path, Query, and Body parameters.

  - Custom Validation Example: Added examples/custom-validation documenting how to implement and register domain-specific validation logic (e.g., SKU formatting).

Changed

  - Validator Middleware: Refactored middleware.Validate to use a package-level defaultValidator instance instead of local instantiation, enabling global configuration and custom tag support.

  - Mux Adapter Integration: Updated examples to use NewMuxAdapter with SimpleCleanerMuxConfig for better compatibility with standard library routing and special characters.


[prerelease] - 2026-02-09

Added

  - New Examples Suite: Added independent, runnable examples for gin-basic, echo-basic, and chi-basic to demonstrate framework integration.

  - Module Patcher Script: Created patch_mods.sh to automate the toggling of replace directives across the monorepo, ensuring golangci-lint compatibility.

  - Mux Adapter: Fully validated and benchmarked the standard library adapter (muxadapter) utilizing Go 1.22+ routing features.

  - Documentation Links: Integrated standard library and local type linking in GoDoc comments using the [Type] syntax.

Changed

  - Testing Modernization: Refactored server_test.go and middleware tests to use t.Context() (Go 1.24+ standard) instead of manual context cancellation.

  - Architecture Refactor: Decomposed massive test suites (RunRouterContract, RunAdvancedRouterContract) into atomic private functions to reduce cognitive complexity (gocognit) and package complexity (cyclop).

  - Validator Middleware: Refactored validator.go to eliminate global variables (gochecknoglobals) by encapsulating the validator instance within the middleware closure.

  - Error Handling: Updated type assertions in formatValidationErrors to use errors.As, ensuring compatibility with wrapped errors (errorlint).

Fixed

  - Linter Compliance: Resolved over 50+ linting issues including golines (line length), goimports (import grouping), noctx (context-aware networking), and nilnesserr (nil pointer safety in tests).

  - Concurrency Safety: Fixed a potential race condition in the server package by implementing net.ListenConfig with context support.

  - Monorepo Path Resolution: Corrected go.mod files in examples to use double replace directives for both core and adapter modules.