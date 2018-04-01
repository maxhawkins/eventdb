// Package prom contains prometheus metrics exported by eventdb.
package prom

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns a handler that exports metrics.
func Handler() http.Handler {
	return promhttp.Handler()
}

// InstrumentHandler decorates an HTTP handler with prometheus metrics jazz.
func InstrumentHandler(name string, handler http.Handler) http.Handler {
	inFlight := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "eventdb_requests_in_flight",
		Help:        "Number of requests currently being served by the handler.",
		ConstLabels: prometheus.Labels{"handler": name},
	})
	promRegister(inFlight)
	handler = promhttp.InstrumentHandlerInFlight(inFlight, handler)

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "eventdb_requests_total",
			Help:        "Total number of requests for the handler.",
			ConstLabels: prometheus.Labels{"handler": name},
		},
		[]string{"code"},
	)
	promRegister(counter)
	handler = promhttp.InstrumentHandlerCounter(counter, handler)

	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "eventdb_response_duration_seconds",
			Help:        "A histogram of request latencies.",
			Buckets:     prometheus.DefBuckets,
			ConstLabels: prometheus.Labels{"handler": name},
		},
		[]string{},
	)
	promRegister(duration)
	handler = promhttp.InstrumentHandlerDuration(duration, handler)

	writeHeaderVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "eventdb_write_header_duration_seconds",
			Help:        "A histogram of time to first write latencies.",
			Buckets:     prometheus.DefBuckets,
			ConstLabels: prometheus.Labels{"handler": name},
		},
		[]string{},
	)
	promRegister(writeHeaderVec)
	handler = promhttp.InstrumentHandlerTimeToWriteHeader(writeHeaderVec, handler)

	responseSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "eventdb_push_request_size_bytes",
			Help:        "A histogram of request sizes for requests.",
			Buckets:     []float64{200, 500, 900, 1500},
			ConstLabels: prometheus.Labels{"handler": name},
		},
		[]string{},
	)
	promRegister(responseSize)
	handler = promhttp.InstrumentHandlerResponseSize(responseSize, handler)

	return handler
}

// HACK(maxhawkins): allow prometheus double-registrations so that the tests
// pass. In the future I should do something better here.
func promRegister(c prometheus.Collector) {
	err := prometheus.Register(c)
	if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
		return
	}
	if err != nil {
		panic(err)
	}
}
