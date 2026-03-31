package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	opsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "heimdall_requests_total",
		Help: "The total number of processed requests",
	}, []string{"type"})

	bytesIngested = promauto.NewCounter(prometheus.CounterOpts{
		Name: "heimdall_ingested_bytes_total",
		Help: "Total bytes written to the cluster",
	})

	inferenceLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "heimdall_ml_inference_latency_seconds",
		Help:    "Latency of the Python ML classification call",
		Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5},
	})
)

func StartMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":2112", nil)
}
