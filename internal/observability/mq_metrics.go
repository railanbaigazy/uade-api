package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	MQPublishTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mq_publish_total",
			Help: "Total number of RabbitMQ publish attempts",
		},
		[]string{"exchange", "routing_key", "result"}, // result=ok|error
	)

	MQConsumeTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mq_consume_total",
			Help: "Total number of RabbitMQ consumed messages",
		},
		[]string{"queue", "result"}, // result=ok|error
	)

	MQAckTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mq_ack_total",
			Help: "Total number of RabbitMQ ACKs",
		},
		[]string{"queue"},
	)

	MQNackTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mq_nack_total",
			Help: "Total number of RabbitMQ NACKs",
		},
		[]string{"queue", "requeue"}, // requeue=true|false
	)

	MQConsumeDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mq_consume_duration_seconds",
			Help:    "Time spent processing a RabbitMQ message",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"queue"},
	)
)
