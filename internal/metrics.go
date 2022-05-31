package internal

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	ResponseTime   prometheus.Histogram
	SuccessCounter prometheus.Counter
	ErrorCounter   prometheus.Counter
}

func MetricHandler(port string) {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":"+port, nil)
}

func NewMetrics(namespace string, collection string, port string) Metrics {
	go MetricHandler(port)
	var cm Metrics
	cm.ResponseTime = promauto.NewHistogram(
		prometheus.HistogramOpts{ //nolint:exhaustivestruct
			Namespace:   namespace,
			Subsystem:   "qsse",
			Name:        "event_response_time",
			Help:        "event response time",
			ConstLabels: prometheus.Labels{"collection": collection},
		},
	)

	cm.SuccessCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "qsse",
			Name:        "event_success_count",
			Help:        "count total success in event operations",
			ConstLabels: prometheus.Labels{"collection": collection},
		},
	)

	cm.ErrorCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "qsse",
			Name:        "event_error_count",
			Help:        "count total errors in event operations",
			ConstLabels: prometheus.Labels{"collection": collection},
		},
	)

	return cm
}

func (m Metrics) AddResponseTime(sample float64) {
	m.ResponseTime.Observe(sample)
}

func (m Metrics) IncErr() {
	m.ErrorCounter.Inc()
}

func (m Metrics) IncSuccess() {
	m.SuccessCounter.Inc()
}
