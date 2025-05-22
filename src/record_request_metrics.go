package pbprometheus

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func SetupRecordRequestMetrics(app core.App, reg *prometheus.Registry, config PBPrometheusConfig) error {
	requestTimer := promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
		Namespace: config.Namespace,
		Name:      "request_duration_seconds",
		Help:      "Duration of requests in seconds",
		Buckets:   config.TimeBuckets,
	}, []string{"collection", "action"})

	app.OnRecordsListRequest().BindFunc(func(req *core.RecordsListRequestEvent) error {
		start := time.Now()
		next := req.Next()
		duration := time.Since(start).Seconds()
		requestTimer.WithLabelValues(req.Collection.Name, "list").Observe(duration)

		return next
	})

	app.OnRecordUpdateRequest().BindFunc(func(req *core.RecordRequestEvent) error {

		start := time.Now()
		next := req.Next()
		duration := time.Since(start).Seconds()
		requestTimer.WithLabelValues(req.Collection.Name, "update").Observe(duration)

		return next
	})

	app.OnRecordCreateRequest().BindFunc(func(req *core.RecordRequestEvent) error {
		start := time.Now()
		next := req.Next()
		duration := time.Since(start).Seconds()
		requestTimer.WithLabelValues(req.Collection.Name, "create").Observe(duration)

		return next
	})

	app.OnRecordDeleteRequest().BindFunc(func(req *core.RecordRequestEvent) error {
		start := time.Now()
		next := req.Next()
		duration := time.Since(start).Seconds()
		requestTimer.WithLabelValues(req.Collection.Name, "delete").Observe(duration)

		return next
	})

	return nil
}
