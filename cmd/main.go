package main

import (
	chiopentelemetry "chi-opentelemetry"
	"chi-opentelemetry/handlers"
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	traceProvider, err := chiopentelemetry.StartTracing()
	if err != nil {
		log.Fatalf("failed to start tracing: %v", err)
	}
	defer func() {
		if err := traceProvider.Shutdown(context.Background()); err != nil {
			log.Fatalf("traceprovider: %v", err)
		}
	}()

	_ = traceProvider.Tracer("chi-telemetry")

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Get("/", Trace(handlers.Hello, "hello-handler"))

	log.Println("Server is running on localhost:3000")
	http.ListenAndServe(":3000", r)
}

// Trace middleware creates a new span for each request
func Trace(f func(w http.ResponseWriter, r *http.Request), span string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sctx, span := chiopentelemetry.Tracer.Start(r.Context(), span)
		defer span.End()

		span.SetAttributes(attribute.KeyValue{
			Key:   "request.path",
			Value: attribute.StringValue(r.URL.Path),
		})
		span.SetAttributes(attribute.KeyValue{
			Key:   "request.method",
			Value: attribute.StringValue(r.Method),
		})
		span.SetAttributes(attribute.KeyValue{
			Key:   "request.remoteAddr",
			Value: attribute.StringValue(r.RemoteAddr),
		})
		span.SetAttributes(attribute.KeyValue{
			Key:   "request.userAgent",
			Value: attribute.StringValue(r.UserAgent()),
		})
		span.SetAttributes(attribute.KeyValue{
			Key:   "request.host",
			Value: attribute.StringValue(r.Host),
		})
		span.SetAttributes(attribute.KeyValue{
			Key:   "request.requestURI",
			Value: attribute.StringValue(r.RequestURI),
		})
		span.SetAttributes(attribute.KeyValue{
			Key:   "request.protocol",
			Value: attribute.StringValue(r.Proto),
		})

		ctx := context.WithValue(r.Context(), "span_ctx", sctx)
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		f(ww, r.WithContext(ctx))

		span.SetAttributes(attribute.KeyValue{
			Key:   "response.status",
			Value: attribute.IntValue(ww.Status()),
		})
	}
}
