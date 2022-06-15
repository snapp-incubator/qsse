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

func NewMetrics(namespace string) Metrics {
	var metric Metrics

	metric.EventCounter = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:exhaustruct
		Namespace: namespace,
		Subsystem: "qsse",
		Name:      "topic_event_count",
		Help:      "count of events in eventsource",
	}, []string{"topic"})

	metric.SubscriberCounter = promauto.NewGaugeVec(prometheus.GaugeOpts{ //nolint:exhaustruct
		Namespace: namespace,
		Subsystem: "qsse",
		Name:      "topic_subscriber_count",
		Help:      "count of topic's subscribers",
	}, []string{"topic"})

	metric.PublishCounter = promauto.NewCounter(prometheus.CounterOpts{ //nolint:exhaustruct
		Namespace: namespace,
		Subsystem: "qsse",
		Name:      "event_publish_total",
		Help:      "count total success published events",
	})

	metric.DistributeCounter = promauto.NewCounter(prometheus.CounterOpts{ //nolint:exhaustruct
		Namespace: namespace,
		Subsystem: "qsse",
		Name:      "event_distribute_total",
		Help:      "count total success distributed events",
	})

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

func (m Metrics) IncPublishEvent() {
	m.PublishCounter.Inc()
}

func (m Metrics) IncDistributeEvent() {
	m.DistributeCounter.Inc()
}
