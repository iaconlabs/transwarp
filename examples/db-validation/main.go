// Package main demonstrates how to perform database-backed validation
// by injecting dependencies into custom validator functions.
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

// MockDatabase simulates a real database connection.
type MockDatabase struct {
	ExistingEmails []string
}

// IsEmailTaken checks if the email already exists in our mock store.
func (db *MockDatabase) IsEmailTaken(email string) bool {
	for _, e := range db.ExistingEmails {
		if strings.EqualFold(e, email) {
			return true
		}
	}
	return false
}

// UserRegistrationDTO defines the input for a new user.
type UserRegistrationDTO struct {
	// The "unique_email" tag is our custom database-backed rule.
	Email    string `json:"email" validate:"required,email,unique_email"`
	Username string `json:"username" validate:"required,min=3"`
}

// EmailUniqueValidator creates a validator function with access to the DB.
// EmailUniqueValidator returns a validator.Func that validates an email string is not already present in the provided MockDatabase.
// The returned function yields `true` if the email is not taken, `false` otherwise.
func EmailUniqueValidator(db *MockDatabase) validator.Func {
	return func(fl validator.FieldLevel) bool {
		email := fl.Field().String()
		// If the email is taken, validation fails (returns false).
		return !db.IsEmailTaken(email)
	}
}

// main initializes a mock database, registers a custom email-uniqueness validator, configures the HTTP routes and middleware, and starts the server on :8080.
//
// It prints startup information and example curl commands demonstrating a failing request for an existing email and a passing request for a new email.
func main() {
	// 1. Initialize our "Database"
	db := &MockDatabase{
		ExistingEmails: []string{"admin@transwarp.io", "user@test.com"},
	}

	// 2. Register the custom validator with the injected DB.
	v := middleware.GetValidator()
	v.RegisterValidation("unique_email", EmailUniqueValidator(db))

	// 3. Setup Transwarp with Mux Adapter
	adp := muxadapter.NewMuxAdapter(nil)

	// 4. Inject the validation middleware into the route
	adp.POST("/register", handleRegister, middleware.Validate(UserRegistrationDTO{}))

	// 5. Start the server
	srv := server.New(server.Config{Addr: ":8080"}, adp)

	fmt.Println("🚀 DB-Backed Validation Example Active")
	fmt.Println("Existing emails: admin@transwarp.io, user@test.com")
	fmt.Println("\n📌 Test TAKE EMAIL (Should Fail):")
	fmt.Println(`curl -X POST http://localhost:8080/register -d '{"email":"admin@transwarp.io", "username":"newbie"}'`)
	fmt.Println("\n📌 Test NEW EMAIL (Should Pass):")
	fmt.Println(`curl -X POST http://localhost:8080/register -d '{"email":"dev@transwarp.io", "username":"transwarper"}'`)

	srv.Start(context.Background())
}

// handleRegister writes a JSON success response containing the validated UserRegistrationDTO extracted from the request context.
// It reads the validated data from r.Context() using router.ValidationKey (expected to hold a *UserRegistrationDTO) and encodes a JSON object with "status", "message", and "data".
func handleRegister(w http.ResponseWriter, r *http.Request) {
	data := r.Context().Value(router.ValidationKey).(*UserRegistrationDTO)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "success",
		"message": "User can be registered",
		"data":    data,
	})
}