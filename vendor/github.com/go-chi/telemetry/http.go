package telemetry

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Gives ability to exposes metrics on a separate http server.
// Example usage to expose `/metrics` endpoint on port `9093` using go std lib http server:
//
//	metricsServer := http.NewServeMux()
//	metricsServer.Handle("/metrics", telemetry.MetricsHandler())
//	err := http.ListenAndServe(":9093", metricsServer)
//
// If you want to also collect http request details as metrics,
// please remember to configure it by setting Config.CollectHttpRequestMetrics to true
func MetricsHandler() http.Handler {
	return reporter.HTTPHandler()
}

// Collector creates a handler that exposes a /metrics endpoint. Passing
// an array of strings to pathPrefixFilters will help reduce the noise on the service
// from random Internet traffic; that is, only the path prefixes will be measured.
func Collector(cfg Config, optPathPrefixFilters ...[]string) func(next http.Handler) http.Handler {
	if (!cfg.AllowAny && !cfg.AllowInternal && !cfg.CollectHttpRequestMetrics) && (cfg.Username == "" || cfg.Password == "") {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	httpMetrics := NewScope("http")

	pathPrefixFilters := []string{}
	if len(optPathPrefixFilters) > 0 {
		pathPrefixFilters = append(pathPrefixFilters, optPathPrefixFilters[0]...)
	}

	authHandler := middleware.BasicAuth(
		"metrics",
		map[string]string{cfg.Username: cfg.Password},
	)

	metricsReporterHandler := chi.Chain(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				// Maybe allow internal traffic
				if cfg.AllowInternal {
					ipAddress := net.ParseIP(getIPAddress(r))
					if isPrivateSubnet(ipAddress) {
						next.ServeHTTP(w, r)
						return
					}
				}

				// Maybw allow basic auth traffic
				if cfg.Username != "" && cfg.Password != "" {
					authHandler(next).ServeHTTP(w, r)
					return
				}

				// Maybe allow any
				if cfg.AllowAny {
					next.ServeHTTP(w, r)
					return
				}

				w.WriteHeader(http.StatusNotFound)
			})
		},
		func(http.Handler) http.Handler {
			// reporter is the global prometheus reporter
			return reporter.HTTPHandler()
		},
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// serve prometheus metrics reporter page
			if r.Method == "GET" && strings.EqualFold(r.URL.Path, "/metrics") {
				metricsReporterHandler.Handler(next).ServeHTTP(w, r)
				return
			}

			// track request mtrics
			//
			// first, check path filters as we skip collecting metrics for these paths
			if len(pathPrefixFilters) > 0 {
				found := false
				for _, p := range pathPrefixFilters {
					if strings.HasPrefix(r.URL.Path, p) {
						found = true
						break
					}
				}
				if !found && r.URL.Path != "/" {
					// skip measurement of the http request
					next.ServeHTTP(w, r)
					return
				}
			}

			// measure request
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			defer httpResponseSample(httpMetrics, time.Now().UTC(), r, ww)

			next.ServeHTTP(ww, r)
		})
	}
}

func httpResponseSample(httpMetrics *Scope, start time.Time, r *http.Request, ww middleware.WrapResponseWriter) {
	status := ww.Status()
	if status == 0 { // TODO: see why we have status = 0 under test conditions (this came up during benchmarks for some of the requests)
		status = http.StatusOK
	}

	// prometheus errors if string is not uft8 encoded
	if !utf8.ValidString(r.URL.Path) {
		return
	}

	metricsKey := fmt.Sprintf("%d %s %s", status, r.Method, r.URL.Path)

	requestMetrics, _ := httpMetrics.GetTaggedScope(metricsKey)
	if requestMetrics == nil {
		requestMetrics = httpMetrics.SetTaggedScope(metricsKey, map[string]string{
			"endpoint": fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			"status":   fmt.Sprintf("%d", status),
		})
	}

	requestMetrics.RecordDuration("request", start, time.Now().UTC())
	requestMetrics.RecordHit("requests")
}
