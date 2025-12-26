package recipe

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Recipe generation metrics
	recipeBuiltDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "eidos_recipe_build_duration_seconds",
			Help:    "Duration of recipe generation in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
		},
	)

	recipeRuleMatchTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eidos_recipe_rule_match_total",
			Help: "Total number of recipe rules matched",
		},
		[]string{"matched"}, // matched or error
	)
)
