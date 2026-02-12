package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/iaconlabs/transwarp/adapter/echoadapter"
	"github.com/iaconlabs/transwarp/middleware"
	"github.com/iaconlabs/transwarp/server"
)

// UserDTO defines the structure for our example request.
type UserDTO struct {
	ID    string `param:"id" validate:"required"`
	Name  string `json:"name" validate:"required,min=3"`
	Email string `json:"email" validate:"required,email"`
}

func main() {

	// 2. Wrap it with the Transwarp Adapter.
	adp := echoadapter.NewEchoAdapter()

	// 3. Define routes using Transwarp's unified API.
	adp.GET("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Group with validation middleware.
	api := adp.Group("/api/v1")
	{
		api.POST("/users/:id", func(w http.ResponseWriter, r *http.Request) {
			// Retrieve validated data from context.
			user := r.Context().Value("transwarp_val").(*UserDTO)

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"message": "User %s created", "data": %+v}`, user.ID, user)
		}, middleware.Validate(UserDTO{}))
	}

	// 4. Start the server using Transwarp's managed server.
	srv := server.New(server.Config{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}, adp)

	fmt.Println("Example server running on http://localhost:8080 over  echo v5")
	if err := srv.Start(context.Background()); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
