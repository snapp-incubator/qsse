package internal

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	EventCounter     *prometheus.GaugeVec
	SubscribeCounter *prometheus.GaugeVec
	PublishCounter   *prometheus.CounterVec
}

func MetricHandler(port string) {
	http.Handle("/metrics", promhttp.Handler())
	log.Println(http.ListenAndServe(":"+port, nil))
}

func NewMetrics(namespace string, port string) Metrics {
	var metric Metrics

	go MetricHandler(port)

	metric.EventCounter = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:exhaustruct
		Namespace: namespace,
		Subsystem: "qsse",
		Name:      "event_source_load",
		Help:      "count of events in eventsource",
	}, []string{"subject"})

	metric.SubscribeCounter = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:exhaustruct
		Namespace: namespace,
		Subsystem: "qsse",
		Name:      "subscribers_count",
		Help:      "count total subscribe in operations",
	}, []string{"subject"})

	metric.PublishCounter = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:exhaustruct
		Namespace: namespace,
		Subsystem: "qsse",
		Name:      "publish_count",
		Help:      "count total success publish in event operations",
	}, []string{"subject"})

	return metric
}

func (n Metrics) IncEvent(subject string) {
	n.EventCounter.With(map[string]string{
		"subject": subject,
	}).Inc()
}

func (n Metrics) DescEvent(subject string) {
	n.EventCounter.With(map[string]string{
		"subject": subject,
	}).Dec()
}

func (n Metrics) IncSubscriber(subject string) {
	n.SubscribeCounter.With(map[string]string{
		"subject": subject,
	}).Inc()
}

func (n Metrics) DescSubscriber(subject string) {
	n.SubscribeCounter.With(map[string]string{
		"subject": subject,
	}).Dec()
}

func (n Metrics) IncPublishEvent(subject string) {
	n.PublishCounter.With(map[string]string{
		"subject": subject,
	}).Inc()
}
