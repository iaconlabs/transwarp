# Changelog

All notable changes to the Transwarp project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.


[Unreleased] - 2026-02-11
Added

    New Examples Suite: Added independent, runnable examples for gin-basic, echo-basic, and chi-basic to demonstrate framework integration.

    Module Patcher Script: Created patch_mods.sh to automate the toggling of replace directives across the monorepo, ensuring golangci-lint compatibility.

    Mux Adapter: Fully validated and benchmarked the standard library adapter (muxadapter) utilizing Go 1.22+ routing features.

    Documentation Links: Integrated standard library and local type linking in GoDoc comments using the [Type] syntax.

Changed

    Testing Modernization: Refactored server_test.go and middleware tests to use t.Context() (Go 1.24+ standard) instead of manual context cancellation.

    Architecture Refactor: Decomposed massive test suites (RunRouterContract, RunAdvancedRouterContract) into atomic private functions to reduce cognitive complexity (gocognit) and package complexity (cyclop).

    Validator Middleware: Refactored validator.go to eliminate global variables (gochecknoglobals) by encapsulating the validator instance within the middleware closure.

    Error Handling: Updated type assertions in formatValidationErrors to use errors.As, ensuring compatibility with wrapped errors (errorlint).

Fixed

    Linter Compliance: Resolved over 50+ linting issues including golines (line length), goimports (import grouping), noctx (context-aware networking), and nilnesserr (nil pointer safety in tests).

    Concurrency Safety: Fixed a potential race condition in the server package by implementing net.ListenConfig with context support.

    Monorepo Path Resolution: Corrected go.mod files in examples to use double replace directives for both core and adapter modules.