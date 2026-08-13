package observability

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"os"
	"time"
)

type Telemetry struct {
	traces   *sdktrace.TracerProvider
	metrics  *sdkmetric.MeterProvider
	Requests metric.Int64Counter
}

func counter(m metric.Meter) metric.Int64Counter {
	c, _ := m.Int64Counter("incidentlab_http_requests_total")
	return c
}

func Setup(ctx context.Context) (*Telemetry, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return &Telemetry{Requests: counter(otel.Meter("incidentlab"))}, nil
	}
	te, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(te))
	otel.SetTracerProvider(tp)
	me, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpoint(endpoint), otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(me, sdkmetric.WithInterval(10*time.Second))))
	otel.SetMeterProvider(mp)
	return &Telemetry{traces: tp, metrics: mp, Requests: counter(mp.Meter("incidentlab"))}, nil
}
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t == nil {
		return nil
	}
	if t.traces != nil {
		if err := t.traces.Shutdown(ctx); err != nil {
			return err
		}
	}
	if t.metrics != nil {
		return t.metrics.Shutdown(ctx)
	}
	return nil
}
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return otel.Tracer("incidentlab").Start(ctx, name)
}
