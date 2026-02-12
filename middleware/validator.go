// Package middleware provides standard net/http middlewares that are compatible
// with any Transwarp adapter.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/iaconlabs/transwarp/adapter"
	"github.com/iaconlabs/transwarp/router"
)

// Internal singleton instance to allow custom tag registration.
var defaultValidator = validator.New()

// GetValidator returns the shared validator instance used by the Validate middleware.
// Use this to register custom validation tags or translations.
func GetValidator() *validator.Validate {
	return defaultValidator
}

// ValidationError represents a specific validation failure for a field.
// It is intended to be returned as part of a structured JSON response.
type ValidationError struct {
	// Field is the name of the struct field that failed validation (json tag preferred).
	Field string `json:"field"`
	// Rule is the name of the validator tag that was violated (e.g., "required", "email").
	Rule string `json:"rule"`
	// Message is a human-readable description of the error.
	Message string `json:"message"`
}

// Validate returns a middleware that performs hybrid binding and validation.
// It unmarshals the JSON body into a new instance of T and maps path parameters
// using the "param" struct tag. If validation fails, it returns a 422 Unprocessable Entity
// with detailed error information. If successful, the validated data is stored
// in the request context under router.ValidationKey.
func Validate[T any](_ T) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Retrieve the Transwarp state injected by the adapter.
			state, ok := r.Context().Value(router.StateKey).(*adapter.TranswarpState)
			if !ok {
				http.Error(w, "Transwarp state not found", http.StatusInternalServerError)
				return
			}

			// 2. Create a new instance of the target type T.
			target := new(T)

			// 3. BINDING: Priority 1 - JSON Body.
			if len(state.Body) > 0 {
				if err := json.Unmarshal(state.Body, target); err != nil {
					sendJSONError(w, "Invalid JSON format", http.StatusBadRequest)
					return
				}
			}

			// 4. BINDING: Priority 2 - Path Parameters (Mapped via "param" tags).
			mapPathParams(target, state.Params)

			// 5. VALIDATION: Execute rules from go-playground/validator.
			if err := defaultValidator.Struct(target); err != nil {
				details := formatValidationErrors(err)
				sendDetailedError(w, details)
				return
			}

			// 6. INJECTION: Store the clean, validated data in the context.
			ctx := context.WithValue(r.Context(), router.ValidationKey, target)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// mapPathParams uses reflection to populate struct fields decorated with the "param" tag
// using values found in the request's path parameters.
func mapPathParams(target any, params map[string]string) {
	val := reflect.ValueOf(target).Elem()
	typ := val.Type()

	for i := range typ.NumField() {
		field := typ.Field(i)
		tag := field.Tag.Get("param")
		if tag != "" {
			if paramVal, exists := params[tag]; exists {
				f := val.Field(i)
				if f.CanSet() && f.Kind() == reflect.String {
					f.SetString(paramVal)
				}
			}
		}
	}
}

// formatValidationErrors converts internal validator errors into a slice of ValidationError.
func formatValidationErrors(err error) []ValidationError {
	var errs []ValidationError
	var vErrors validator.ValidationErrors

	// Fix: Use errors.As instead of a direct type assertion to support wrapped errors.
	if errors.As(err, &vErrors) {
		for _, vErr := range vErrors {
			errs = append(errs, ValidationError{
				Field:   strings.ToLower(vErr.Field()),
				Rule:    vErr.Tag(),
				Message: createMsgForTag(vErr),
			})
		}
	}
	return errs
}

// createMsgForTag generates a professional error message based on the failed validation tag.
func createMsgForTag(v validator.FieldError) string {
	switch v.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return fmt.Sprintf("Minimum length/value is %s", v.Param())
	case "max":
		return fmt.Sprintf("Maximum length/value is %s", v.Param())
	default:
		return fmt.Sprintf("Validation failed on rule: %s", v.Tag())
	}
}

// sendJSONError sends a simple structured JSON error message.
func sendJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// sendDetailedError sends a 422 response containing a list of validation errors.
func sendDetailedError(w http.ResponseWriter, errors []ValidationError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "error",
		"errors": errors,
	})
}
