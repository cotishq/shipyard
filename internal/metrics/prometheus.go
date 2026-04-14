package metrics

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)


var (
	// Gauges
    QueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "shipyard_queue_depth",
        Help: "Number of deployments waiting to be processed",
    })

    ActiveBuilds = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "shipyard_active_builds",
        Help: "Number of deployments currently building",
    })

	// Counters
    DeploymentSuccessTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "shipyard_deployment_success_total",
        Help: "Total number of successful deployments",
    }, []string{"project_id"})

    DeploymentFailureTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "shipyard_deployment_failure_total",
        Help: "Total number of failed deployments",
    }, []string{"project_id"})

    RetryTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "shipyard_retry_total",
        Help: "Total number of deployment retries",
    }, []string{"project_id"})

	// Histogram for build duration
    BuildDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "shipyard_build_duration_seconds",
        Help:    "Build duration in seconds",
        Buckets: []float64{10, 30, 60, 120, 300, 600, 1200},
    }, []string{"project_id", "build_preset"})


)
// UpdateGauges queries the database and updates gauge metrics
func UpdateGauges(db *sql.DB) error {
    // Queue depth
    var queueCount int
    err := db.QueryRow(`
        SELECT COUNT(*) FROM deployments WHERE status = 'QUEUED'
    `).Scan(&queueCount)
    if err != nil {
        return err
    }
    QueueDepth.Set(float64(queueCount))

    // Active builds
    var buildCount int
    err = db.QueryRow(`
        SELECT COUNT(*) FROM deployments WHERE status = 'BUILDING'
    `).Scan(&buildCount)
    if err != nil {
        return err
    }
    ActiveBuilds.Set(float64(buildCount))

    return nil
}
// RecordDeploymentSuccess increments success counter
func RecordDeploymentSuccess(projectID string) {
    DeploymentSuccessTotal.WithLabelValues(projectID).Inc()
}

// RecordDeploymentFailure increments failure counter
func RecordDeploymentFailure(projectID string) {
    DeploymentFailureTotal.WithLabelValues(projectID).Inc()
}

// RecordRetry increments retry counter
func RecordRetry(projectID string) {
    RetryTotal.WithLabelValues(projectID).Inc()
}

// RecordBuildDuration records build duration
func RecordBuildDuration(projectID, buildPreset string, durationSeconds float64) {
    BuildDurationSeconds.WithLabelValues(projectID, buildPreset).Observe(durationSeconds)
}