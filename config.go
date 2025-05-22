// Package pbprometheus provides Prometheus metrics integration for PocketBase applications.
// It allows monitoring of HTTP requests, record operations, collection counts,
// realtime connections, and standard Go runtime/process metrics.
package pbprometheus

import (
	"fmt"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PBPrometheusConfig struct {
	MetricsPath                 string `default:"/metrics"`
	DisableRecordRequestMetrics bool
	DisableRecordCountStats     bool
	DisableGeneralHttpMetrics   bool
	DisableRealtimeMetrics      bool
	RecordCountCronSchedule     string    `default:"*/5 * * * *"`
	Namespace                   string    `default:"pocketbase"`
	TimeBuckets                 []float64 `default:"[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]"`
}

func (config *PBPrometheusConfig) SetDefaults() {
	if config.MetricsPath == "" {
		config.MetricsPath = "/metrics" // Default
	}

	if config.Namespace == "" {
		config.Namespace = "pocketbase" // Default
	}

	if config.RecordCountCronSchedule == "" {
		config.RecordCountCronSchedule = "*/5 * * * *" // Default
	}

	if config.TimeBuckets == nil {
		config.TimeBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10} // Default
	}
}

func Setup(app core.App, config PBPrometheusConfig) error { // Return error

	config.SetDefaults() // Set default values
	// Use a dedicated registry for better control
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	if !config.DisableRecordRequestMetrics {
		if err := SetupRecordRequestMetrics(app, reg, config); err != nil { // Pass reg and config
			return fmt.Errorf("failed to setup record request metrics: %w", err)
		}
	}

	if !config.DisableRecordCountStats {
		if err := SetupRecordCountStats(app, reg, config); err != nil { // Pass reg and config
			return fmt.Errorf("failed to setup record count stats: %w", err)
		}
	}

	if !config.DisableGeneralHttpMetrics {
		// Setup general HTTP metrics (highly recommended)
		if err := SetupGeneralHttpMetrics(app, reg, config); err != nil { // Pass reg and config
			return fmt.Errorf("failed to setup general HTTP metrics: %w", err)
		}
	}

	if !config.DisableRealtimeMetrics {
		if err := SetupRealtimeConnectionMonitoring(app, reg, config); err != nil { // Pass reg and config
			return fmt.Errorf("failed to setup realtime connection monitoring: %w", err)
		}
	}

	metricsPath := config.MetricsPath
	if metricsPath == "" {
		metricsPath = "/metrics" // Default
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET(metricsPath, apis.WrapStdHandler(promhttp.HandlerFor(reg, promhttp.HandlerOpts{})))
		app.Logger().Info(fmt.Sprintf("Prometheus metrics endpoint registered at %s", metricsPath))
		return se.Next()
	})
	app.Logger().Info("Prometheus exporter setup complete.")
	return nil
}
