# PocketBase Prometheus Exporter (pbprometheus)

[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/pbprometheus)](https://goreportcard.com/report/github.com/yourusername/pbprometheus)
[![GoDoc](https://godoc.org/github.com/yourusername/pbprometheus?status.svg)](https://godoc.org/github.com/yourusername/pbprometheus)
<!-- Add build status badge once GitHub Actions CI is set up -->
<!-- [![Build Status](https://github.com/yourusername/pbprometheus/actions/workflows/go.yml/badge.svg)](https://github.com/yourusername/pbprometheus/actions/workflows/go.yml) -->
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`pbprometheus` is a Go package that seamlessly integrates Prometheus metrics into your [PocketBase](https://pocketbase.io/) application. It provides valuable insights into your application's performance, request handling, database operations, and realtime connections.

## Features

*   **General HTTP Metrics:** Tracks total HTTP requests, request duration histograms, and status codes for all incoming requests (excluding the metrics endpoint itself).
*   **PocketBase Record Operations:** Monitors the duration of Create, Read (List), Update, and Delete (CRUD) operations on your collections.
*   **Collection Record Counts:** Periodically reports the total number of records in each collection.
*   **Realtime Connection Monitoring:** Tracks the number of active Realtime (SSE) connections.
*   **Go Runtime Metrics:** Includes standard Go process metrics (goroutines, memory, GC, etc.) via `collectors.NewGoCollector()`.
*   **Process Metrics:** Includes standard process metrics (CPU, memory, file descriptors, etc.) via `collectors.NewProcessCollector()`.
*   **Configurable:** Allows customization of the metrics endpoint path, Prometheus namespace, metric time buckets, and more.
*   **Easy Integration:** Simple setup with a single function call.

## Installation

To install `pbprometheus`, use `go get`:

```bash
go get github.com/yourusername/pbprometheus
```

Make sure your project is using Go Modules.

## Usage

Integrating `pbprometheus` into your PocketBase application is straightforward. Here's a basic example of how to use it in your `main.go` or wherever you initialize your PocketBase app:

```go
package main

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	// Import the pbprometheus package
	"github.com/yourusername/pbprometheus"
)

func main() {
	app := pocketbase.New()

	// --- Initialize pbprometheus ---
	// Create a configuration (or use defaults)
	promConfig := pbprometheus.PBPrometheusConfig{
		// MetricsPath:                 "/custom-metrics", // Optional: Default is "/metrics"
		// Namespace:                   "myapp_pocketbase",  // Optional: Default is "pocketbase"
		// DisableRecordRequestMetrics: false,
		// DisableRecordCountStats:     false,
		// DisableGeneralHttpMetrics:   false,
		// DisableRealtimeMetrics:      false,
		// RecordCountCronSchedule:     "*/10 * * * *", // Optional: Default is "*/5 * * * *"
		// TimeBuckets:                 []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}, // Optional
	}

	// Setup Prometheus metrics
	// The Setup function will register all metric collectors and the HTTP handler.
	if err := pbprometheus.Setup(app, promConfig); err != nil {
		log.Fatalf("Failed to setup Prometheus exporter: %v", err)
	}
	// --- End of pbprometheus initialization ---

	// Add your other PocketBase configurations and hooks here...

	// Add migrate command
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: true, // auto creates migration files when making collection changes
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
```

After starting your PocketBase application, Prometheus metrics will be available at the configured `MetricsPath` (default: `/metrics`).

## Configuration

The `pbprometheus.Setup` function accepts a `PBPrometheusConfig` struct to customize its behavior:

```go
type PBPrometheusConfig struct {
	// MetricsPath is the HTTP path where Prometheus metrics will be exposed.
	// Default: "/metrics"
	MetricsPath string `default:"/metrics"`

	// DisableRecordRequestMetrics, if true, disables metrics for record create, update, delete, and list operations.
	// Default: false
	DisableRecordRequestMetrics bool

	// DisableRecordCountStats, if true, disables the cron job that periodically counts records in collections.
	// Default: false
	DisableRecordCountStats bool

	// DisableGeneralHttpMetrics, if true, disables general HTTP request metrics (total requests, duration).
	// Default: false
	DisableGeneralHttpMetrics bool

	// DisableRealtimeMetrics, if true, disables monitoring of active realtime (SSE) connections.
	// Default: false
	DisableRealtimeMetrics bool

	// RecordCountCronSchedule is the cron schedule for updating record count stats.
	// Default: "*/5 * * * *" (every 5 minutes)
	RecordCountCronSchedule string `default:"*/5 * * * *"`

	// Namespace is the Prometheus namespace prefix for all metrics exposed by this package.
	// Default: "pocketbase"
	Namespace string `default:"pocketbase"`

	// TimeBuckets defines the buckets for histogram metrics (e.g., request duration).
	// Default: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10] (seconds)
	TimeBuckets []float64 `default:"[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]"`
}
```

If a field is not set in the `PBPrometheusConfig` you provide, its default value will be used.

## Exposed Metrics (Examples)

All metrics are prefixed with the configured `Namespace` (default: `pocketbase`).

*   **General HTTP Metrics:**
    *   `{namespace}_http_requests_total{method, path, status_code}` (Counter)
    *   `{namespace}_http_request_duration_seconds{method, path}` (Histogram)
*   **Record Operation Metrics:**
    *   `{namespace}_request_duration_seconds{collection, action}` (Histogram) - where `action` can be `list`, `create`, `update`, `delete`.
*   **Record Count Stats:**
    *   `{namespace}_record_count{collection}` (Gauge)
*   **Realtime Metrics:**
    *   `{namespace}_realtime_connections_active` (Gauge)
*   **Go Runtime & Process Metrics:**
    *   `go_goroutines`, `go_memstats_alloc_bytes`, etc. (from `collectors.NewGoCollector()`)
    *   `process_cpu_seconds_total`, `process_resident_memory_bytes`, etc. (from `collectors.NewProcessCollector()`)

You can scrape these metrics by configuring your Prometheus server to target the `MetricsPath` on your PocketBase application's host and port.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request or open an Issue for bugs, feature requests, or improvements.

When contributing, please ensure:
1.  Code is well-formatted (`go fmt`).
2.  Tests are added or updated for new features or bug fixes.
3.  The `README.md` is updated if necessary.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.