package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type Labels struct {
	Kind           string
	Name           string
	IdlingResource string
	Namespace      string
}

var (
	kindLabel           = "kind"
	nameLabel           = "name"
	idlingResourceLabel = "idlingresource"
	namespaceLabel      = "namespace"
	labelNames          = []string{kindLabel, nameLabel, idlingResourceLabel, namespaceLabel}
)

func (l *Labels) ToPrometheusLabels() prometheus.Labels {
	return prometheus.Labels{
		kindLabel:           l.Kind,
		nameLabel:           l.Name,
		idlingResourceLabel: l.IdlingResource,
		namespaceLabel:      l.Namespace,
	}
}

var (
	IdleCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kidle_idle_phase_total",
			Help: "Number of idle phase",
		},
		labelNames,
	)
	IdleGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kidle_idle_phase_gauge",
			Help: "Number of idle phase",
		},
		labelNames,
	)
	WakeupCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kidle_wakeup_phase_total",
			Help: "Number of wakeup phase",
		},
		labelNames,
	)
	WakeupGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kidle_wakeup_phase_gauge",
			Help: "Number of wakeup phase",
		},
		labelNames,
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(IdleCount, WakeupCount, IdleGauge, WakeupGauge)
}
