package adapter

import (
	"regexp"
	"strings"
)

// colonRegex identifies parameter placeholders in the format ":name" (e.g., :id, :user-id).
var colonRegex = regexp.MustCompile(`:([a-zA-Z0-9._-]+)`)

// TranslatePath converts Transwarp-style path parameters (:param) into
// brace-style placeholders ({param}). This is commonly used for adapters
// like go-chi or [http.ServeMux] that expect the latter format.
func TranslatePath(path string) string {
	return colonRegex.ReplaceAllStringFunc(path, func(m string) string {
		key := strings.TrimPrefix(m, ":")
		return "{" + key + "}"
	})
}
