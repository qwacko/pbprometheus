package pbprometheus

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// --- responseWriterInterceptor definition (as before) ---
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func newResponseWriterInterceptor(w http.ResponseWriter) *responseWriterInterceptor {
	return &responseWriterInterceptor{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rwi *responseWriterInterceptor) WriteHeader(code int) {
	if !rwi.wroteHeader {
		rwi.statusCode = code
		rwi.wroteHeader = true
	}
	rwi.ResponseWriter.WriteHeader(code)
}

func (rwi *responseWriterInterceptor) Write(b []byte) (int, error) {
	return rwi.ResponseWriter.Write(b)
}

func (rwi *responseWriterInterceptor) Status() int {
	return rwi.statusCode
}

// --- End of responseWriterInterceptor ---

func SetupGeneralHttpMetrics(app core.App, reg *prometheus.Registry, config PBPrometheusConfig) error {
	httpRequestsTotal := promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
		Namespace: config.Namespace,
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests.",
	}, []string{"method", "path", "status_code"})

	httpRequestDuration := promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
		Namespace: config.Namespace,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latencies in seconds.",
		Buckets:   config.TimeBuckets,
	}, []string{"method", "path"})

	// 1. Define the standard net/http middleware function
	metricsStdMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == config.MetricsPath { // Don't instrument metrics endpoint
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			interceptor := newResponseWriterInterceptor(w)

			defer func() {
				duration := time.Since(start)
				status := interceptor.Status()
				requestPath := r.URL.Path // High cardinality risk still applies

				if queryIdx := strings.Index(requestPath, "?"); queryIdx != -1 {
					requestPath = requestPath[:queryIdx]
				}
				if requestPath == "" {
					requestPath = "/"
				}

				httpRequestsTotal.WithLabelValues(
					r.Method,
					requestPath,
					strconv.Itoa(status),
				).Inc()

				httpRequestDuration.WithLabelValues(
					r.Method,
					requestPath,
				).Observe(duration.Seconds())
			}()

			next.ServeHTTP(interceptor, r)
		})
	}

	// 2. Register the wrapped standard middleware using app.OnServe()
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// 3. Wrap the standard net/http middleware
		// apis.WrapStdMiddleware returns *hook.Handler[*core.RequestEvent]
		pbWrappedMiddleware := apis.WrapStdMiddleware(metricsStdMiddleware)

		// 4. Bind the wrapped middleware globally to the router
		// se.Router.Bind() expects one or more *hook.Handler[*core.RequestEvent]
		se.Router.BindFunc(pbWrappedMiddleware)

		app.Logger().Info("General HTTP metrics middleware registered via OnServe and WrapStdMiddleware.")
		return se.Next() // Ensure other OnServe hooks can run
	})

	return nil
}
