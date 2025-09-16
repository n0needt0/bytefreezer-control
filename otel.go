package main

import (
	"context"
	"time"

	"github.com/n0needt0/bytefreezer-control/config"
	"github.com/n0needt0/go-goodies/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// InitOtelProvider initializes OpenTelemetry providers
func InitOtelProvider(conf *config.Config) func() {
	ctx := context.Background()

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(conf.Otel.ServiceName),
			semconv.ServiceVersionKey.String(conf.App.Version),
		),
	)
	if err != nil {
		log.Errorf("Failed to create OTEL resource: %v", err)
		return func() {}
	}

	var cleanupFunctions []func()

	// Initialize tracing
	if traceProvider, cleanup := initTracing(ctx, res, conf); traceProvider != nil {
		otel.SetTracerProvider(traceProvider)
		cleanupFunctions = append(cleanupFunctions, cleanup)
	}

	// Initialize metrics
	if meterProvider, cleanup := initMetrics(ctx, res, conf); meterProvider != nil {
		otel.SetMeterProvider(meterProvider)
		cleanupFunctions = append(cleanupFunctions, cleanup)
	}

	// Return cleanup function
	return func() {
		for _, cleanup := range cleanupFunctions {
			if cleanup != nil {
				cleanup()
			}
		}
	}
}

// initTracing initializes the tracing provider
func initTracing(ctx context.Context, res *resource.Resource, conf *config.Config) (*sdktrace.TracerProvider, func()) {
	// Create OTLP trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(conf.Otel.Endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Errorf("Failed to create OTLP trace exporter: %v", err)
		return nil, nil
	}

	// Create trace provider
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := traceProvider.Shutdown(ctx); err != nil {
			log.Errorf("Failed to shutdown trace provider: %v", err)
		}
	}

	return traceProvider, cleanup
}

// initMetrics initializes the metrics provider
func initMetrics(ctx context.Context, res *resource.Resource, conf *config.Config) (metric.MeterProvider, func()) {
	// Create OTLP metric exporter
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(conf.Otel.Endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		log.Errorf("Failed to create OTLP metric exporter: %v", err)
		return nil, nil
	}

	// Create metrics provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				metricExporter,
				sdkmetric.WithInterval(time.Duration(conf.Otel.ScrapeIntervalSeconds)*time.Second),
			),
		),
	)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := meterProvider.Shutdown(ctx); err != nil {
			log.Errorf("Failed to shutdown meter provider: %v", err)
		}
	}

	return meterProvider, cleanup
}
