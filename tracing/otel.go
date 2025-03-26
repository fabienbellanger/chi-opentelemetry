package tracing

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	otelTrace "go.opentelemetry.io/otel/trace"
)

const otelScopeName = "chi-telemetry"

var (
	tracer = otel.Tracer(otelScopeName, otelTrace.WithInstrumentationVersion("1.0.0"))
)

func StartTracing() (*trace.TracerProvider, error) {
	headers := map[string]string{
		"content-type": "application/json",
	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint("localhost:4318"),
			otlptracehttp.WithHeaders(headers),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating new exporter: %w", err)
	}

	tracerprovider := trace.NewTracerProvider(
		trace.WithBatcher(
			exporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultScheduleDelay*time.Millisecond),
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("chi-telemetry"),
			),
		),
	)

	otel.SetTracerProvider(tracerprovider)

	return tracerprovider, nil
}

func NewSpanFromContext(ctx context.Context) otelTrace.Span {
	span := otelTrace.SpanFromContext(ctx)

	_, file, line, ok := runtime.Caller(1)
	if ok {
		span.SetAttributes(attribute.KeyValue{
			Key:   semconv.CodeFilepathKey,
			Value: attribute.StringValue(file),
		})
		span.SetAttributes(attribute.KeyValue{
			Key:   semconv.CodeLineNumberKey,
			Value: attribute.IntValue(line),
		})
	}

	return span
}

func NewSpan(ctx context.Context, name string) (context.Context, otelTrace.Span) {
	sctx, span := tracer.Start(ctx, name)

	_, file, line, ok := runtime.Caller(1)
	if ok {
		span.SetAttributes(attribute.KeyValue{
			Key:   semconv.CodeFilepathKey,
			Value: attribute.StringValue(file),
		})
		span.SetAttributes(attribute.KeyValue{
			Key:   semconv.CodeLineNumberKey,
			Value: attribute.IntValue(line),
		})
	}

	return sctx, span
}

// Trace middleware creates a new span for each request
func Trace(f func(w http.ResponseWriter, r *http.Request), span string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sctx, span := tracer.Start(r.Context(), span)
		defer span.End()

		_, file, line, ok := runtime.Caller(0)
		if ok {
			span.SetAttributes(attribute.KeyValue{
				Key:   semconv.CodeFilepathKey,
				Value: attribute.StringValue(file),
			})
			span.SetAttributes(attribute.KeyValue{
				Key:   semconv.CodeLineNumberKey,
				Value: attribute.IntValue(line),
			})
		}

		span.SetAttributes(attribute.KeyValue{
			Key:   semconv.URLPathKey,
			Value: attribute.StringValue(r.URL.Path),
		})
		span.SetAttributes(attribute.KeyValue{
			Key:   semconv.HTTPRequestMethodKey,
			Value: attribute.StringValue(r.Method),
		})
		span.SetAttributes(attribute.KeyValue{
			Key:   "http.remoteAddr",
			Value: attribute.StringValue(r.RemoteAddr),
		})
		span.SetAttributes(attribute.KeyValue{
			Key:   semconv.UserAgentNameKey,
			Value: attribute.StringValue(r.UserAgent()),
		})
		span.SetAttributes(attribute.KeyValue{
			Key:   semconv.HostNameKey,
			Value: attribute.StringValue(r.Host),
		})
		span.SetAttributes(attribute.KeyValue{
			Key:   semconv.URLFullKey,
			Value: attribute.StringValue(r.RequestURI),
		})
		span.SetAttributes(semconv.NetworkProtocolName(r.Proto))

		ctx := context.WithValue(r.Context(), "span_ctx", sctx)
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		f(ww, r.WithContext(ctx))

		span.SetAttributes(semconv.HTTPResponseStatusCode(ww.Status()))
	}
}
