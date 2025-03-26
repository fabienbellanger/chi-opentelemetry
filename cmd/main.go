package main

import (
	"chi-opentelemetry/handlers"
	"chi-opentelemetry/tracing"
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	traceProvider, err := tracing.StartTracing()
	if err != nil {
		log.Fatalf("failed to start tracing: %v", err)
	}
	defer func() {
		if err := traceProvider.Shutdown(context.Background()); err != nil {
			log.Fatalf("traceprovider: %v", err)
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", tracing.Trace(handlers.Hello, "hello-handler"))

	log.Println("Server is running on localhost:3000")
	http.ListenAndServe(":3000", r)
}
