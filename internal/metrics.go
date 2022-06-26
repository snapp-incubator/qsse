package internal

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	EventCounter      *prometheus.GaugeVec
	SubscriberCounter *prometheus.GaugeVec
	PublishCounter    prometheus.Counter
	DistributeCounter prometheus.Counter
}

func NewMetrics(namespace, subSystem string) Metrics {
	var metric Metrics

	metric.EventCounter = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:exhaustruct
		Namespace: namespace,
		Subsystem: subSystem,
		Name:      "event_count",
		Help:      "count of events in eventsource",
	}, []string{"topic"})

	metric.SubscriberCounter = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:exhaustruct
		Namespace: namespace,
		Subsystem: subSystem,
		Name:      "subscriber_count",
		Help:      "count of topic's subscribers",
	}, []string{"topic"})

	return metric
}

func (m Metrics) IncEvent(topic string) {
	m.EventCounter.With(map[string]string{
		"topic": topic,
	}).Inc()
}

func (m Metrics) DecEvent(topic string) {
	m.EventCounter.With(map[string]string{
		"topic": topic,
	}).Dec()
}

func (m Metrics) IncSubscriber(topic string) {
	m.SubscriberCounter.With(map[string]string{
		"topic": topic,
	}).Inc()
}

func (m Metrics) DecSubscriber(topic string) {
	m.SubscriberCounter.With(map[string]string{
		"topic": topic,
	}).Dec()
}
