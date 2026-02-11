// Package adapter contains internal logic and state structures used to bridge
// standard net/http behavior with third-party web frameworks.
package adapter

import "strings"

// TranswarpState centralizes request metadata such as route parameters and
// the request body to avoid redundant context allocations.
type TranswarpState struct {
	// Params holds a normalized map of path parameters.
	Params map[string]string
	// Body stores a cached version of the request body for multiple reads.
	Body []byte
}

// Clone creates a new string instance from the input to prevent race conditions
// when strings are modified across different goroutines.
func Clone(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(s)
	return b.String()
}
