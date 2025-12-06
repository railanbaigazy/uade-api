package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	MQPublishedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "uade_mq_messages_published_total",
			Help: "Total number of MQ messages published",
		},
		[]string{"routing_key"},
	)

	MQPublishErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "uade_mq_publish_errors_total",
			Help: "Total number of MQ publish errors",
		},
		[]string{"routing_key"},
	)
)

func Register() {
	prometheus.MustRegister(MQPublishedTotal, MQPublishErrorsTotal)
}
