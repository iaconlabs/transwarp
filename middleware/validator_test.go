package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/profe-ajedrez/transwarp/adapter"
	"github.com/profe-ajedrez/transwarp/middleware"
	"github.com/profe-ajedrez/transwarp/router"
)

// SignupRequest defines a test structure that demonstrates hybrid binding
// from both path parameters and the JSON request body.
type SignupRequest struct {
	ID    string `param:"id" validate:"required,min=5"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"required,gte=18"`
}

// TestValidate_HybridBinding verifies that data is correctly merged from
// multiple sources into a single validated struct.
func TestValidate_HybridBinding(t *testing.T) {
	// 1. Setup: Create the middleware for our struct.
	mw := middleware.Validate(SignupRequest{})

	// finalHandler checks if the injected data is present in the context.
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := r.Context().Value(router.ValidationKey).(*SignupRequest)
		if !ok {
			t.Fatal("Validated data not found in context")
		}

		if data.ID != "user-123" {
			t.Errorf("Path ID not mapped correctly. Expected user-123, got %s", data.ID)
		}
		if data.Email != "test@transwarp.io" {
			t.Errorf("Email from JSON not mapped correctly")
		}
		w.WriteHeader(http.StatusOK)
	})

	// 2. Simulate the state typically injected by a Transwarp adapter.
	state := &adapter.TranswarpState{
		Params: map[string]string{"id": "user-123"},
		Body:   []byte(`{"email": "test@transwarp.io", "age": 25}`),
	}

	// 3. Execute request using modern testing context.
	req := httptest.NewRequest(http.MethodPost, "/users/user-123", bytes.NewReader(state.Body))
	ctx := context.WithValue(t.Context(), router.StateKey, state)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mw(finalHandler).ServeHTTP(rr, req)

	// 4. Verification.
	if rr.Code != http.StatusOK {
		t.Errorf("Expected StatusOK, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

// TestValidate_ValidationError ensures the middleware blocks invalid requests
// and returns a structured 422 error response.
func TestValidate_ValidationError(t *testing.T) {
	mw := middleware.Validate(SignupRequest{})

	// State with invalid data (short ID, malformed email, underage).
	state := &adapter.TranswarpState{
		Params: map[string]string{"id": "123"}, // min=5 will fail
		Body:   []byte(`{"email": "not-an-email", "age": 10}`),
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	ctx := context.WithValue(t.Context(), router.StateKey, state)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// The final handler should never execute.
	mw(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("Final handler should not have been executed due to validation errors")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected 422, got %d", rr.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	errList, ok := response["errors"].([]any)
	if !ok || len(errList) != 3 {
		t.Errorf("Expected 3 validation errors, got %d", len(errList))
	}
}
