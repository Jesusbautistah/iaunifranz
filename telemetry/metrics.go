package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	namespace = "ollama"
)

type Metrics struct {
	Requests metric.Int64Counter
}

func NewMetrics(meter metric.Meter) *Metrics {
	req, _ := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("The total number of requests on the endpoints."),
		metric.WithUnit("requests"),
	)

	return &Metrics{
		Requests: req,
	}
}

func (m *Metrics) RecordRequests(ctx context.Context, action string, statusCode int64, status string) {
	m.Requests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("action", action),
		attribute.Int64("status_code", statusCode),
		attribute.String("status", status),
	))
}

func NewPrometheusMeterProvider() (*sdkmetric.MeterProvider, error) {
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(namespace),
			semconv.ServiceVersionKey.String("v0.1.0"),
		),
		resource.WithProcessRuntimeDescription(),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(exporter),
	)

	// Start go runtime metric collection.
	err = runtime.Start(runtime.WithMeterProvider(meterProvider),
		runtime.WithMinimumReadMemStatsInterval(time.Second))
	if err != nil {
		return nil, err
	}

	return meterProvider, nil
}

func InitMetrics() (*Metrics, error) {
	mp, err := NewPrometheusMeterProvider()
	if err != nil {
		return nil, err
	}
	otel.SetMeterProvider(mp)

	meter := mp.Meter(namespace, metric.WithInstrumentationVersion(runtime.Version()))
	return NewMetrics(meter), nil
}
