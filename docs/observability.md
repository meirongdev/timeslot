# Observability Design (OpenTelemetry)

> **Status**: OpenTelemetry integration is currently **not implemented** and is listed here for future consideration.

To integrate Timeslot into a larger microservices ecosystem, we employ **OpenTelemetry (OTel)** for tracing, metrics, and potentially logging. This ensures the service is observable and its performance can be analyzed using tools like Jaeger, Tempo, Prometheus, or Honeycomb.

---

## 1. Tracing Design

Tracing helps visualize the request lifecycle, from the public API to the internal SQLite database.

### Core Instrumentation Points
- **HTTP Middleware**: All incoming requests (API and Admin) are wrapped with OTel instrumentation to extract trace contexts from headers.
- **Database (SQLite)**: Use an OTel-compliant database wrapper (e.g., `github.com/uptrace/opentelemetry-go-extra/otelsql`) to trace all SQL queries.
- **iCal Sync Worker**: Each sync job is treated as a root span to track performance and errors during external iCal fetching.

### Span Context Propagation
- Support W3C Trace Context and Baggage propagation headers to maintain trace continuity when called from or calling other services.

---

## 2. Metrics Design

Metrics provide visibility into the service's health and resource usage.

### Key Metrics
- **HTTP**: Request counts, latency histograms, and error rates per endpoint.
- **Sync Worker**: Last sync timestamp, sync duration, and number of events parsed.
- **Slot Stats**: Available slots count.
- **Go Runtime**: Memory usage, CPU, goroutines, and GC stats (via default OTel Go metrics).

---

## 3. Configuration (`config.json`)

The observability layer is configured via the OTLP (OpenTelemetry Protocol) exporter.

```json
{
  "otel": {
    "enabled": true,
    "service_name": "timeslot",
    "service_version": "v0.1.0",
    "endpoint": "otel-collector.monitoring.svc:4317",
    "insecure": true,
    "protocol": "grpc",
    "sampling_ratio": 1.0
  }
}
```

---

## 4. Implementation Strategy

### Initialization (`backend/main.go`)
At startup, initialize the OTel SDK, set the Global Tracer Provider, and the Global Text Map Propagator.

### HTTP Middleware
```go
func (h *Handler) withOTel(next http.HandlerFunc) http.HandlerFunc {
    // Use go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
    return otelhttp.NewHandler(next, "api-handler").ServeHTTP
}
```

### Database Wrapper (`backend/db/db.go`)
```go
// Replace standard driver with OTel-instrumented one
db, err := otelsql.Open("sqlite3", dsn, otelsql.WithAttributes(
    semconv.DBSystemSqlite,
))
```

---

## Implementation Roadmap

| Feature | Description | Status |
|---|---|---|
| **OTLP Exporter** | Send traces and metrics to an OTel Collector via OTLP. | Not Implemented |
| **HTTP Instrumentation** | Wrap `net/http` handlers with trace context extraction. | Not Implemented |
| **SQL Instrumentation** | Wrap SQLite driver to capture query spans. | Not Implemented |
| **Prometheus Exporter** | Optional: Expose metrics on `/metrics` endpoint for Prometheus scraping. | Not Implemented |

---

## Benefits
- **Full Traceability**: See exactly why a sync is slow or an API call fails.
- **Service Dependency Mapping**: Automatically visualize how Timeslot fits into your HomeLab's microservice mesh.
- **Proactive Alerting**: Set up alerts based on latency or error rate thresholds in Grafana/Prometheus.
