package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type Labels struct {
	Kind           string
	Name           string
	Idlingresource string
	Namespace      string
}

func (l *Labels) ToPrometheusLabels() prometheus.Labels {
	return prometheus.Labels{
		"kind":           l.Kind,
		"name":           l.Name,
		"idlingresource": l.Idlingresource,
		"namespace":      l.Namespace,
	}
}

var (
	IdleCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kidle_idle_phase_total",
			Help: "Number of idle phase",
		},
		[]string{
			"kind",
			"name",
			"idlingresource",
			"namespace",
		},
	)
	WakeupCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kidle_wakeup_phase_total",
			Help: "Number of wakeup phase",
		},
		[]string{
			"kind",
			"name",
			"idlingresource",
			"namespace",
		},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(IdleCount, WakeupCount)
}
