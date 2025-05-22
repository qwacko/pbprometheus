package pbprometheus

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func SetupRealtimeConnectionMonitoring(app core.App, reg *prometheus.Registry, config PBPrometheusConfig) error {
	// Setup a gauge to track the number of active WebSocket connections

	realtimeConnectionsActive := promauto.With(reg).NewGauge(prometheus.GaugeOpts{
		Namespace: "pocketbase",
		Name:      "realtime_connections_active",
		Help:      "Number of active Realtime (SSE) connections.",
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		app.OnRealtimeConnectRequest().BindFunc(func(e *core.RealtimeConnectRequestEvent) error {
			realtimeConnectionsActive.Inc()

			e.Next()
			realtimeConnectionsActive.Dec()

			return nil
		})

		return se.Next()
	})

	return nil
}
