package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Infrastructure label keys reused across counters. Business metrics live in
// the caller's package — define them there to avoid coupling the template to
// a particular domain vocabulary.
const (
	labelReason = "reason"
	labelJob    = "job"
	labelResult = "result"
)

// Infrastructure counter vectors — populated by initInfraMetrics during Setup.
// Initialized with a throwaway registry in init() so callers (e.g. tests
// without Setup) don't hit nil pointer panics.
var (
	RedisCommandsTotal            *prometheus.CounterVec
	RedisCommandDuration          *prometheus.HistogramVec
	KafkaMessagesSentTotal        *prometheus.CounterVec
	KafkaSendDuration             *prometheus.HistogramVec
	CronJobExecutionsTotal        *prometheus.CounterVec
	CronJobDuration               *prometheus.HistogramVec
	CronJobLastExecutionTimestamp *prometheus.GaugeVec
	CronJobSkippedTotal           *prometheus.CounterVec
)

func init() {
	nopReg := prometheus.NewRegistry()
	if err := initInfraMetrics(nopReg, nil); err != nil {
		panic("metrics: init infra metrics: " + err.Error())
	}
}

var (
	cronBuckets  = []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120, 300}
	redisBuckets = []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1}
)

func initInfraMetrics(reg prometheus.Registerer, constLabels prometheus.Labels) error {
	RedisCommandsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "app_redis_commands_total", Help: "Total Redis command executions.",
		ConstLabels: constLabels,
	}, []string{"command", labelResult})
	RedisCommandDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "app_redis_command_duration_seconds", Help: "Redis command latency.",
		Buckets: redisBuckets, ConstLabels: constLabels,
	}, []string{"command"})

	KafkaMessagesSentTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "app_kafka_messages_sent_total", Help: "Total Kafka messages sent.",
		ConstLabels: constLabels,
	}, []string{"topic", labelResult})
	KafkaSendDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "app_kafka_send_duration_seconds", Help: "Kafka message send latency.",
		Buckets: defaultLatencyBuckets, ConstLabels: constLabels,
	}, []string{"topic"})

	CronJobExecutionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "app_cronjob_executions_total", Help: "Total cron job executions.",
		ConstLabels: constLabels,
	}, []string{labelJob, labelResult})
	CronJobDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "app_cronjob_duration_seconds", Help: "Cron job execution latency.",
		Buckets: cronBuckets, ConstLabels: constLabels,
	}, []string{labelJob})
	CronJobLastExecutionTimestamp = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "app_cronjob_last_execution_timestamp", Help: "Unix timestamp of last cron job execution.",
		ConstLabels: constLabels,
	}, []string{labelJob})
	CronJobSkippedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "app_cronjob_skipped_total", Help: "Total skipped cron job executions (re-entrancy guard).",
		ConstLabels: constLabels,
	}, []string{labelJob})

	for _, c := range []prometheus.Collector{
		RedisCommandsTotal, RedisCommandDuration,
		KafkaMessagesSentTotal, KafkaSendDuration,
		CronJobExecutionsTotal, CronJobDuration, CronJobLastExecutionTimestamp, CronJobSkippedTotal,
	} {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}
