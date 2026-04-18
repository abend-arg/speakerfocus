package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "speakerfocus"

var latencyBuckets = []float64{
	0.001,
	0.0025,
	0.005,
	0.01,
	0.025,
	0.05,
	0.1,
	0.25,
	0.5,
	1.0,
	2.5,
	5.0,
}

type Metrics struct {
	chunkDuration   *prometheus.HistogramVec
	endToEndLatency *prometheus.HistogramVec
	stageDuration   *prometheus.HistogramVec
	chunksTotal     *prometheus.CounterVec
	vadDecisions    *prometheus.CounterVec
	stageErrors     *prometheus.CounterVec
}

func NewMetrics(registry *prometheus.Registry) *Metrics {
	m := &Metrics{
		chunkDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "chunk_duration_seconds",
			Help:      "Wall-clock processing duration for one audio chunk.",
			Buckets:   latencyBuckets,
		}, []string{"result"}),
		endToEndLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "end_to_end_latency_seconds",
			Help:      "Wall-clock latency from chunk capture to sink write completion.",
			Buckets:   latencyBuckets,
		}, []string{"result"}),
		stageDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "stage_duration_seconds",
			Help:      "Wall-clock latency for one pipeline stage.",
			Buckets:   latencyBuckets,
		}, []string{"stage"}),
		chunksTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "chunks_total",
			Help:      "Total audio chunks processed by result.",
		}, []string{"result"}),
		vadDecisions: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "vad_decisions_total",
			Help:      "Total VAD policy decisions by action and state.",
		}, []string{"action", "state"}),
		stageErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "stage_errors_total",
			Help:      "Total pipeline stage errors by stage and reason.",
		}, []string{"stage", "reason"}),
	}

	registry.MustRegister(
		m.chunkDuration,
		m.endToEndLatency,
		m.stageDuration,
		m.chunksTotal,
		m.vadDecisions,
		m.stageErrors,
	)

	return m
}

func (m *Metrics) ObserveChunkDuration(result string, duration time.Duration) {
	m.chunkDuration.WithLabelValues(result).Observe(duration.Seconds())
}

func (m *Metrics) ObserveEndToEndLatency(result string, duration time.Duration) {
	m.endToEndLatency.WithLabelValues(result).Observe(duration.Seconds())
}

func (m *Metrics) ObserveStageDuration(stage string, duration time.Duration) {
	m.stageDuration.WithLabelValues(stage).Observe(duration.Seconds())
}

func (m *Metrics) IncChunk(result string) {
	m.chunksTotal.WithLabelValues(result).Inc()
}

func (m *Metrics) IncVADDecision(action string, state string) {
	m.vadDecisions.WithLabelValues(action, state).Inc()
}

func (m *Metrics) IncStageError(stage string, reason string) {
	m.stageErrors.WithLabelValues(stage, reason).Inc()
}

func NewRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewBuildInfoCollector(),
	)
	return registry
}
