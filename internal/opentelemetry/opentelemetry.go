package opentelemetry

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"os"
	"runtime"
)

func Init(ctx context.Context) (func(), error) {
	// Avoid logging errors when local OpenTelemetry Collector is not available, for example:
	// "failed to upload metrics: [...]: dial tcp 127.0.0.1:4318: connect: connection refused"
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		// do nothing
	}))

	// Work around https://github.com/open-telemetry/opentelemetry-go/issues/4834
	if _, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT"); !ok {
		if err := os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"); err != nil {
			return nil, err
		}
	}

	defaultResource := resource.Default()

	customResource, err := resource.New(ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithOSType(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.HostArchKey.String(runtime.GOARCH),
		),
		resource.WithAttributes(
			semconv.ServiceName("cirrus-cli"),
			semconv.ServiceVersion(version.Version),
		),
	)
	if err != nil {
		return nil, err
	}

	resource, err := resource.Merge(defaultResource, customResource)
	if err != nil {
		return nil, err
	}

	// Metrics
	var finalizers []func()

	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, err
	}
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)
	finalizers = append(finalizers, func() {
		_ = meterProvider.Shutdown(ctx)
	})
	otel.SetMeterProvider(meterProvider)

	// Traces
	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource),
		sdktrace.WithBatcher(traceExporter),
	)
	finalizers = append(finalizers, func() {
		_ = traceProvider.Shutdown(ctx)
	})
	otel.SetTracerProvider(traceProvider)

	return func() {
		for _, finalizer := range finalizers {
			finalizer()
		}
	}, nil
}
