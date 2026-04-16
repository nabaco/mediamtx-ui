package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	AuthCallbacks = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mediamtx_ui_auth_callbacks_total",
		Help: "Total mediamtx auth callback requests, by outcome.",
	}, []string{"action", "allowed"})

	ActiveStreams = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mediamtx_ui_active_streams",
		Help: "Number of currently active (ready) mediamtx streams.",
	})

	APIRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mediamtx_ui_api_requests_total",
		Help: "Total UI API requests, by method and path.",
	}, []string{"method", "path", "status"})

	LoginAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mediamtx_ui_login_attempts_total",
		Help: "Total login attempts, by outcome.",
	}, []string{"outcome"}) // "success" | "failure"

	StreamReaders = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mediamtx_ui_stream_readers",
		Help: "Number of current readers per stream.",
	}, []string{"stream"})
)
