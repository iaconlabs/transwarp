package transwarp

import (
	"fmt"
	log "log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery returns a middleware that recovers from panics, logs the error,
// and returns an Internal Server Error (500) to the client.
// If stack is true, it includes the stack trace in the log and response.
func Recovery(stack bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					message := fmt.Sprintf("PANIC RECOVERED: %v", err)
					if stack {
						message = fmt.Sprintf("%s\n\n%s", message, string(debug.Stack()))
					}
					log.Info(message)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)

					body := `{"error": "Internal Server Error"}`
					if stack {
						body = fmt.Sprintf(`{"error": %q}`, message)
					}
					_, _ = w.Write([]byte(body))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
