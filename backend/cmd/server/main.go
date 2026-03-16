package main

import (
	"context"
	"log"
	"net/http"

	"analytics-backend/internal/app"
)

func main() {
	ctx := context.Background()

	application, err := app.New(ctx)
	if err != nil {
		log.Fatalf("startup failed: %v", err)
	}
	defer application.Close()

	application.StartMaintenance(ctx)

	address := ":" + application.Config.Port
	log.Printf("listening on %s", address)
	if err := http.ListenAndServe(address, application.Router().Routes()); err != nil {
		log.Fatalf("listen failed: %v", err)
	}
}
