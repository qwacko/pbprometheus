package pbprometheus

import (
	"bufio"
	"errors" // For custom error in Hijack
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	// No more echo import needed here
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// responseWriterInterceptor intercepts the HTTP status code and implements common net/http interfaces.
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func newResponseWriterInterceptor(
	w http.ResponseWriter,
) *responseWriterInterceptor {
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
	n, err := rwi.ResponseWriter.Write(b)
	return n, err
}

func (rwi *responseWriterInterceptor) Status() int {
	return rwi.statusCode
}

// Flush implements the http.Flusher interface.
// Crucial for SSE (Server-Sent Events) used by PocketBase realtime.
func (rwi *responseWriterInterceptor) Flush() {
	if flusher, ok := rwi.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
	// If the underlying writer doesn't support Flush, this is a no-op.
	// SSE might fail if the original writer didn't support it and was expected to.
}

// Hijack implements the http.Hijacker interface.
// Allows taking control of the underlying connection.
// This is how one might get to call SetWriteDeadline on the net.Conn.
func (rwi *responseWriterInterceptor) Hijack() (
	net.Conn,
	*bufio.ReadWriter,
	error,
) {
	if hijacker, ok := rwi.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, errors.New(
		"http.Hijacker not supported by underlying ResponseWriter",
	)
}

// Push implements the http.Pusher interface (for HTTP/2 server push).
func (rwi *responseWriterInterceptor) Push(
	target string,
	opts *http.PushOptions,
) error {
	if pusher, ok := rwi.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	// http.ErrNotSupported is the standard error to return if Push is not supported.
	return http.ErrNotSupported
}

// --- End of responseWriterInterceptor ---

func SetupGeneralHttpMetrics(
	app core.App,
	reg *prometheus.Registry,
	config PBPrometheusConfig,
) error {
	httpRequestsTotal := promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status_code"},
	)

	httpRequestDuration := promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latencies in seconds.",
			Buckets:   config.TimeBuckets,
		},
		[]string{"method", "path"},
	)

	logger := app.Logger()

	// This is a standard net/http middleware constructor
	metricsStdMiddleware := func(next http.Handler) http.Handler {
		// This is the actual net/http handler
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == config.MetricsPath {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			interceptor := newResponseWriterInterceptor(w) // Use enhanced interceptor

			defer func() {
				if rec := recover(); rec != nil {
					logger.Error(
						"PANIC in HTTP metrics middleware defer block",
						"error",
						rec,
						"method",
						r.Method,
						"path",
						r.URL.Path,
					)
					if !interceptor.wroteHeader {
						// Use the interceptor to write the error, so status is captured
						http.Error(
							interceptor,
							http.StatusText(http.StatusInternalServerError),
							http.StatusInternalServerError,
						)
					}
					// Note: After a panic, the connection might be in an unstable state.
					// Writing an error is best effort.
				}

				duration := time.Since(start)
				status := interceptor.Status()
				method := r.Method
				requestPath := r.URL.Path

				if queryIdx := strings.Index(requestPath, "?"); queryIdx != -1 {
					requestPath = requestPath[:queryIdx]
				}
				if requestPath == "" {
					requestPath = "/"
				}

				httpRequestsTotal.WithLabelValues(
					method,
					requestPath,
					strconv.Itoa(status),
				).Inc()

				httpRequestDuration.WithLabelValues(method, requestPath).Observe(
					duration.Seconds(),
				)
			}()

			next.ServeHTTP(interceptor, r) // Pass the enhanced interceptor
		})
	}

	// app.OnServe() and se.Router.BindFunc(apis.WrapStdMiddleware(...))
	// is the PocketBase way to integrate net/http middleware.
	// This part remains correct.
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		pbWrappedMiddleware := apis.WrapStdMiddleware(metricsStdMiddleware)
		se.Router.BindFunc(pbWrappedMiddleware) // Binds it to PocketBase's router
		logger.Info("General HTTP metrics middleware registered.")
		return se.Next()
	})

	return nil
}
