package handlers

import (
	"chi-opentelemetry/tracing"
	"context"
	"errors"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func Hello(w http.ResponseWriter, r *http.Request) {
	sctx := r.Context().Value("span_ctx").(context.Context)
	name := r.URL.Query().Get("name")

	time.Sleep(50 * time.Millisecond)
	res, err := formatHello(sctx, name)
	if err != nil {
		span := tracing.NewSpanFromContext(sctx)
		span.SetStatus(codes.Error, err.Error())
		span.AddEvent("query parameter 'name' is empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Write(res)
}

func formatHello(ctx context.Context, name string) ([]byte, error) {
	_, span := tracing.NewSpan(ctx, "formatHello")
	defer span.End()

	span.SetAttributes(attribute.KeyValue{
		Key:   "name",
		Value: attribute.StringValue(name),
	})

	time.Sleep(100 * time.Millisecond)

	if name == "" {
		return []byte(""), errors.New("name is empty")
	}
	return []byte("Hello " + name + "!"), nil
}
